package mongodb

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type bulkWriter struct {
	col           *mongo.Collection
	documentLimit uint16 // https://www.mongodb.com/docs/manual/reference/limits/#mongodb-limit-Write-Command-Batch-Limit-Size - must be <= 100000
	throttleTime  time.Duration
	flushTimeout  time.Duration
	logger        log.Logger

	done    chan struct{}
	trigger chan bool

	mutex  sync.Mutex
	models map[string]*store.SigningRecord
	wg     sync.WaitGroup
}

func newBulkWriter(col *mongo.Collection, documentLimit uint16, throttleTime time.Duration, flushTimeout time.Duration, logger log.Logger) *bulkWriter {
	r := &bulkWriter{
		col:           col,
		documentLimit: documentLimit,
		throttleTime:  throttleTime,
		flushTimeout:  flushTimeout,
		done:          make(chan struct{}),
		trigger:       make(chan bool, 1),
		logger:        logger,
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.run()
	}()
	return r
}

func toSigningRecordFilter(signingRecord *store.SigningRecord) bson.M {
	res := bson.M{"_id": signingRecord.GetId()}
	return res
}

func getSigningRecordCreationDate(defaultTime time.Time, signingRecord *store.SigningRecord) int64 {
	ret := defaultTime.UTC().UnixNano()
	if signingRecord.GetCredential().GetDate() > 0 && signingRecord.GetCredential().GetDate() < ret {
		ret = signingRecord.GetCredential().GetDate()
	}
	return ret
}

func setValueByDate(key, datePath string, dateOperator string, date int64, value interface{}) bson.M {
	return bson.M{
		"$set": bson.M{
			key: bson.M{
				"$ifNull": bson.A{
					bson.M{
						"$cond": bson.M{
							"if": bson.M{
								dateOperator: bson.A{"$" + datePath, date},
							},
							"then": value,
							"else": "$" + key,
						},
					}, value,
				},
			},
		},
	}
}

func updateSigningRecord(signingRecord *store.SigningRecord) []bson.M {
	creationDate := signingRecord.GetCreationDate()
	if creationDate == 0 {
		creationDate = getSigningRecordCreationDate(time.Now(), signingRecord)
	}
	ret := []bson.M{
		{"$set": bson.M{
			"_id":               signingRecord.GetId(),
			store.CommonNameKey: signingRecord.GetCommonName(),
			store.OwnerKey:      signingRecord.GetOwner(),
			store.PublicKeyKey:  signingRecord.GetPublicKey(),
		}},
	}
	ret = append(ret, setValueByDate(store.CreationDateKey, store.CreationDateKey, "$gt", creationDate, creationDate))
	if signingRecord.GetCredential() != nil {
		ret = append(ret, setValueByDate(store.CredentialKey, store.CredentialKey+"."+store.DateKey, "$lt", signingRecord.GetCredential().GetDate(), signingRecord.GetCredential()))
	}
	return ret
}

func convertSigningRecordToWriteModel(signingRecord *store.SigningRecord) mongo.WriteModel {
	return mongo.NewUpdateOneModel().SetFilter(toSigningRecordFilter(signingRecord)).SetUpdate(updateSigningRecord(signingRecord)).SetUpsert(true)
}

func mergeLatestUpdateSigningRecord(toUpdate *store.SigningRecord, latest *store.SigningRecord) *store.SigningRecord {
	if toUpdate == nil {
		return latest
	}
	if latest.GetCredential().GetDate() > toUpdate.GetCredential().GetDate() {
		toUpdate.Credential = latest.GetCredential()
	}
	if latest.GetCreationDate() < toUpdate.GetCreationDate() {
		toUpdate.CreationDate = latest.GetCreationDate()
		if toUpdate.GetCommonName() == "" {
			toUpdate.CommonName = latest.GetCommonName()
		}
		if toUpdate.GetOwner() == "" {
			toUpdate.Owner = latest.GetOwner()
		}
		if toUpdate.GetPublicKey() == "" {
			toUpdate.PublicKey = latest.GetPublicKey()
		}
	}
	return toUpdate
}

func (b *bulkWriter) popSigningRecords() map[string]*store.SigningRecord {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	models := b.models
	b.models = nil
	return models
}

func (b *bulkWriter) Push(signingRecords ...*store.SigningRecord) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.models == nil {
		b.models = make(map[string]*store.SigningRecord)
	}
	for _, signingRecord := range signingRecords {
		b.models[signingRecord.GetId()] = mergeLatestUpdateSigningRecord(b.models[signingRecord.GetId()], signingRecord)
	}
	select {
	case b.trigger <- true:
	default:
	}
}

func (b *bulkWriter) numSigningRecords() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return len(b.models)
}

func (b *bulkWriter) run() {
	ticker := time.NewTicker(b.throttleTime)
	tickerRunning := true
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if b.tryBulkWrite() == 0 && tickerRunning {
				ticker.Stop()
				tickerRunning = false
			}
		case <-b.trigger:
			if !tickerRunning {
				ticker.Reset(b.throttleTime)
				tickerRunning = true
			}
			if b.numSigningRecords() > int(b.documentLimit) {
				b.tryBulkWrite()
			}
		case <-b.done:
			return
		}
	}
}

func (b *bulkWriter) bulkWrite() (int, error) {
	SigningRecords := b.popSigningRecords()
	if len(SigningRecords) == 0 {
		return 0, nil
	}
	ctx := context.Background()
	if b.flushTimeout != 0 {
		ctx1, cancel := context.WithTimeout(context.Background(), b.flushTimeout)
		defer cancel()
		ctx = ctx1
	}
	m := make([]mongo.WriteModel, 0, int(b.documentLimit)+1)

	var errors *multierror.Error
	for _, SigningRecord := range SigningRecords {
		m = append(m, convertSigningRecordToWriteModel(SigningRecord))
		if b.documentLimit == 0 || len(m)%int(b.documentLimit) == 0 {
			_, err := b.col.BulkWrite(ctx, m, options.BulkWrite().SetOrdered(false))
			if err != nil {
				errors = multierror.Append(errors, err)
			}
			m = m[:0]
		}
	}

	if len(m) > 0 {
		_, err := b.col.BulkWrite(ctx, m, options.BulkWrite().SetOrdered(false))
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	return len(SigningRecords), errors.ErrorOrNil()
}

func (b *bulkWriter) tryBulkWrite() int {
	n, err := b.bulkWrite()
	if err != nil {
		b.logger.Errorf("failed to bulk update Signing records: %w", err)
	}
	return n
}

func (b *bulkWriter) Close() {
	close(b.done)
	b.wg.Wait()
	b.tryBulkWrite()
}
