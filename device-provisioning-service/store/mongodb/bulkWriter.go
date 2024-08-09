package mongodb

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
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

	closed  atomic.Bool
	done    chan struct{}
	trigger chan bool

	mutex  sync.Mutex
	models map[string]*store.ProvisioningRecord
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

func toProvisioningRecordFilter(provisionedDevice *store.ProvisioningRecord) bson.M {
	res := bson.M{"_id": provisionedDevice.GetId()}
	return res
}

func getProvisioningRecordCreationDate(defaultTime time.Time, provisionedDevice *store.ProvisioningRecord) int64 {
	ret := defaultTime.UTC().UnixNano()
	if provisionedDevice.GetAttestation().GetDate() > 0 && provisionedDevice.GetAttestation().GetDate() < ret {
		ret = provisionedDevice.GetAttestation().GetDate()
	}
	if provisionedDevice.GetCloud().GetStatus().GetDate() > 0 && provisionedDevice.GetCloud().GetStatus().GetDate() < ret {
		ret = provisionedDevice.GetCloud().GetStatus().GetDate()
	}
	if provisionedDevice.GetAcl().GetStatus().GetDate() > 0 && provisionedDevice.GetAcl().GetStatus().GetDate() < ret {
		ret = provisionedDevice.GetAcl().GetStatus().GetDate()
	}
	if provisionedDevice.GetCredential().GetStatus().GetDate() > 0 && provisionedDevice.GetCredential().GetStatus().GetDate() < ret {
		ret = provisionedDevice.GetCredential().GetStatus().GetDate()
	}
	if provisionedDevice.GetOwnership().GetStatus().GetDate() > 0 && provisionedDevice.GetOwnership().GetStatus().GetDate() < ret {
		ret = provisionedDevice.GetOwnership().GetStatus().GetDate()
	}
	if provisionedDevice.GetPlgdTime().GetDate() > 0 && provisionedDevice.GetPlgdTime().GetDate() < ret {
		ret = provisionedDevice.GetPlgdTime().GetDate()
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

func updateProvisioningRecord(provisionedDevice *store.ProvisioningRecord) []bson.M {
	creationDate := provisionedDevice.GetCreationDate()
	if creationDate == 0 {
		creationDate = getProvisioningRecordCreationDate(time.Now(), provisionedDevice)
	}
	ret := []bson.M{
		{"$set": bson.M{
			"_id":                      provisionedDevice.GetId(),
			store.EnrollmentGroupIDKey: provisionedDevice.GetEnrollmentGroupId(),
			store.DeviceIDKey:          provisionedDevice.GetDeviceId(),
			store.LocalEndpointsKey:    provisionedDevice.GetLocalEndpoints(),
			store.OwnerKey:             provisionedDevice.GetOwner(),
		}},
	}
	ret = append(ret, setValueByDate(store.CreationDateKey, store.CreationDateKey, "$gt", creationDate, creationDate))
	if provisionedDevice.GetAttestation() != nil {
		ret = append(ret, setValueByDate(store.AttestationKey, store.AttestationKey+"."+store.DateKey, "$lt", provisionedDevice.GetAttestation().GetDate(), provisionedDevice.GetAttestation()))
	}
	if provisionedDevice.GetCloud() != nil {
		ret = append(ret, setValueByDate(store.CloudKey, store.CloudKey+"."+store.StatusKey+"."+store.DateKey, "$lt", provisionedDevice.GetCloud().GetStatus().GetDate(), provisionedDevice.GetCloud()))
	}
	if provisionedDevice.GetAcl() != nil {
		ret = append(ret, setValueByDate(store.ACLKey, store.ACLKey+"."+store.StatusKey+"."+store.DateKey, "$lt", provisionedDevice.GetAcl().GetStatus().GetDate(), provisionedDevice.GetAcl()))
	}
	if provisionedDevice.GetCredential() != nil {
		ret = append(ret, setValueByDate(store.CredentialKey, store.CredentialKey+"."+store.StatusKey+"."+store.DateKey, "$lt", provisionedDevice.GetCredential().GetStatus().GetDate(), provisionedDevice.GetCredential()))
	}
	if provisionedDevice.GetOwnership() != nil {
		ret = append(ret, setValueByDate(store.OwnershipKey, store.OwnershipKey+"."+store.StatusKey+"."+store.DateKey, "$lt", provisionedDevice.GetOwnership().GetStatus().GetDate(), provisionedDevice.GetOwnership()))
	}
	if provisionedDevice.GetPlgdTime() != nil {
		ret = append(ret, setValueByDate(store.PlgdTimeKey, store.PlgdTimeKey+"."+store.DateKey, "$lt", provisionedDevice.GetPlgdTime().GetDate(), provisionedDevice.GetPlgdTime()))
	}
	return ret
}

func convertProvisioningRecordToWriteModel(provisionedDevice *store.ProvisioningRecord) mongo.WriteModel {
	return mongo.NewUpdateOneModel().SetFilter(toProvisioningRecordFilter(provisionedDevice)).SetUpdate(updateProvisioningRecord(provisionedDevice)).SetUpsert(true)
}

func setEmptyField(field *string, value string) {
	if *field == "" {
		*field = value
	}
}

func setNonEmptyValue(field *string, value string) {
	if value != "" {
		*field = value
	}
}

func mergeLatestUpdateProvisioningRecord(toUpdate *store.ProvisioningRecord, latest *store.ProvisioningRecord) *store.ProvisioningRecord {
	if toUpdate == nil {
		return latest
	}

	setEmptyField(&toUpdate.Owner, latest.GetOwner())
	setEmptyField(&toUpdate.EnrollmentGroupId, latest.GetEnrollmentGroupId())
	setEmptyField(&toUpdate.DeviceId, latest.GetDeviceId())

	if len(toUpdate.GetLocalEndpoints()) == 0 {
		toUpdate.LocalEndpoints = latest.GetLocalEndpoints()
	}
	if latest.GetAttestation().GetDate() > toUpdate.GetAttestation().GetDate() {
		toUpdate.Attestation = latest.GetAttestation()
	}
	if latest.GetCloud().GetStatus().GetDate() > toUpdate.GetCloud().GetStatus().GetDate() {
		toUpdate.Cloud = latest.GetCloud()
	}
	if latest.GetAcl().GetStatus().GetDate() > toUpdate.GetAcl().GetStatus().GetDate() {
		toUpdate.Acl = latest.GetAcl()
	}
	if latest.GetCredential().GetStatus().GetDate() > toUpdate.GetCredential().GetStatus().GetDate() {
		toUpdate.Credential = latest.GetCredential()
	}
	if latest.GetOwnership().GetStatus().GetDate() > toUpdate.GetOwnership().GetStatus().GetDate() {
		toUpdate.Ownership = latest.GetOwnership()
	}
	if latest.GetPlgdTime().GetDate() > toUpdate.GetPlgdTime().GetDate() {
		toUpdate.PlgdTime = latest.GetPlgdTime()
	}
	if latest.GetCreationDate() < toUpdate.GetCreationDate() {
		toUpdate.CreationDate = latest.GetCreationDate()
		setNonEmptyValue(&toUpdate.EnrollmentGroupId, latest.GetEnrollmentGroupId())
		setNonEmptyValue(&toUpdate.Owner, latest.GetOwner())
		setNonEmptyValue(&toUpdate.DeviceId, latest.GetDeviceId())
		if len(latest.GetLocalEndpoints()) > 0 {
			toUpdate.LocalEndpoints = latest.GetLocalEndpoints()
		}
	}
	return toUpdate
}

func (b *bulkWriter) popProvisioningRecords() map[string]*store.ProvisioningRecord {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	models := b.models
	b.models = nil
	return models
}

func (b *bulkWriter) push(provisioningRecords ...*store.ProvisioningRecord) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.models == nil {
		b.models = make(map[string]*store.ProvisioningRecord)
	}
	for _, provisioningRecord := range provisioningRecords {
		b.models[provisioningRecord.GetId()] = mergeLatestUpdateProvisioningRecord(b.models[provisioningRecord.GetId()], provisioningRecord)
	}
	select {
	case b.trigger <- true:
	default:
	}
}

func (b *bulkWriter) numProvisioningRecords() int {
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
			if b.numProvisioningRecords() > int(b.documentLimit) {
				b.tryBulkWrite()
			}
		case <-b.done:
			return
		}
	}
}

func (b *bulkWriter) bulkWrite() (int, error) {
	provisioningRecords := b.popProvisioningRecords()
	if len(provisioningRecords) == 0 {
		return 0, nil
	}
	ctx := context.Background()
	if b.flushTimeout != 0 {
		ctx1, cancel := context.WithTimeout(context.Background(), b.flushTimeout)
		defer cancel()
		ctx = ctx1
	}
	m := make([]mongo.WriteModel, 0, int(b.documentLimit)+1)

	var errors []error
	for _, provisioningRecord := range provisioningRecords {
		m = append(m, convertProvisioningRecordToWriteModel(provisioningRecord))
		if b.documentLimit == 0 || len(m)%int(b.documentLimit) == 0 {
			_, err := b.col.BulkWrite(ctx, m, options.BulkWrite().SetOrdered(false))
			if err != nil {
				errors = append(errors, err)
			}
			m = m[:0]
		}
	}

	if len(m) > 0 {
		_, err := b.col.BulkWrite(ctx, m, options.BulkWrite().SetOrdered(false))
		if err != nil {
			errors = append(errors, err)
		}
	}
	var err error
	if len(errors) == 1 {
		err = errors[0]
	} else if len(errors) > 0 {
		err = fmt.Errorf("%v", errors)
	}
	return len(provisioningRecords), err
}

func (b *bulkWriter) tryBulkWrite() int {
	n, err := b.bulkWrite()
	if err != nil {
		b.logger.Errorf("failed to bulk update provisioning records: %w", err)
	}
	return n
}

func (b *bulkWriter) Close() {
	if !b.closed.CompareAndSwap(false, true) {
		return
	}
	close(b.done)
	b.wg.Wait()
	b.tryBulkWrite()
}
