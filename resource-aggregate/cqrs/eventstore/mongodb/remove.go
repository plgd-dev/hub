package mongodb

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// remove all documents which has latest version less than the version
func (s *EventStore) removeDocumentsUpToVersion(ctx context.Context, queries []eventstore.VersionQuery) error {
	var errors *multierror.Error
	queryResolver := newQueryResolver(signOperator_lt)
	for _, q := range queries {
		err := queryResolver.set(q)
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot remove events version for query('%+v'): %w", q, err))
			continue
		}
	}

	filter, hint := queryResolver.toMongoQuery(latestVersionKey)
	opts := options.Delete()
	opts.SetHint(hint)
	_, err := s.client.Database(s.DBName()).Collection(getEventCollectionName()).DeleteMany(ctx, filter, opts)
	if err != nil {
		errors = multierror.Append(errors, err)
	}
	return errors.ErrorOrNil()
}

// pull events from documents which has first version less than the version
func (s *EventStore) removeEventsUpToVersion(ctx context.Context, queries []eventstore.VersionQuery) error {
	var errors *multierror.Error
	for _, q := range queries {
		queryResolver := newQueryResolver(signOperator_lt)
		err := queryResolver.set(q)
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot pull events version for query('%+v'): %w", q, err))
			continue
		}
		filter, hint := queryResolver.toMongoQuery(firstVersionKey)
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
			errors = multierror.Append(errors, err)
			continue
		}
	}
	return errors.ErrorOrNil()
}

// RemoveUpToVersion deletes the aggregated events up to a specific version.
func (s *EventStore) RemoveUpToVersion(ctx context.Context, versionQueries []eventstore.VersionQuery) error {
	normalizedVersionQueries := make(map[string][]eventstore.VersionQuery)
	for _, query := range versionQueries {
		normalizedVersionQueries[query.GroupID] = append(normalizedVersionQueries[query.GroupID], query)
	}

	var errors *multierror.Error
	for _, queries := range normalizedVersionQueries {
		if err := s.removeDocumentsUpToVersion(ctx, queries); err != nil {
			errors = multierror.Append(errors, err)
			continue
		}

		if err := s.removeEventsUpToVersion(ctx, queries); err != nil {
			errors = multierror.Append(errors, err)
			continue
		}
	}
	return errors.ErrorOrNil()
}
