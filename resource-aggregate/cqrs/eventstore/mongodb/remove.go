package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
)

// RemoveUpToVersion deletes the aggregates events up to a specific version.
func (s *EventStore) RemoveUpToVersion(ctx context.Context, versionQueries []eventstore.VersionQuery) error {
	normalizedVersionQueries := make(map[string][]eventstore.VersionQuery)
	for _, query := range versionQueries {
		normalizedVersionQueries[query.GroupID] = append(normalizedVersionQueries[query.GroupID], query)
	}

	var errors []error
	for _, queries := range normalizedVersionQueries {
		queryResolver := newQueryResolver(signOperator_lt)
		for _, q := range queries {
			err := queryResolver.set(q)
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot remove events version for query('%+v'): %w", q, err))
				continue
			}
		}
		// remove all documents which has latest version less than the version
		filter, hint := queryResolver.toMongoQuery(latestVersionKey)
		opts := options.Delete()
		opts.SetHint(hint)
		_, err := s.client.Database(s.DBName()).Collection(getEventCollectionName()).DeleteMany(ctx, filter, opts)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		// pull events from documents which has first version less than the version
		for _, q := range queries {
			queryResolver := newQueryResolver(signOperator_lt)
			err := queryResolver.set(q)
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot pull events version for query('%+v'): %w", q, err))
				continue
			}
			filter, hint = queryResolver.toMongoQuery(firstVersionKey)
			updOpts := options.Update()
			updOpts.SetHint(hint)
			_, err = s.client.Database(s.DBName()).Collection(getEventCollectionName()).UpdateMany(ctx, filter, bson.M{
				"$set": bson.M{
					firstVersionKey: q.Version,
				},
				"$pull": bson.M{
					eventsKey: bson.M{
						versionKey: bson.M{
							"$lt": q.Version,
						},
					},
				},
			}, updOpts)
			if err != nil {
				errors = append(errors, err)
				continue
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}
	return nil
}
