package mongo

import (
	"errors"
	"strings"
	"sync"

	"github.com/logistics-id/engine/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// FilterSearch builds a case-insensitive `$or` regex filter for the specified fields.
//
// Example:
//
//	FilterSearch("john", "name", "email")
//	=>
//	{
//	  "$or": [
//	    {"name": {"$regex": "john", "$options": "i"}},
//	    {"email": {"$regex": "john", "$options": "i"}}
//	  ]
//	}
func FilterSearch(search string, fields ...string) primitive.M {
	var orConditions []bson.M

	for _, field := range fields {
		orConditions = append(orConditions, bson.M{
			field: bson.M{
				"$regex":   search,
				"$options": "i",
			},
		})
	}

	return bson.M{"$or": orConditions}
}

// RequestSort converts a list of sort keys into a MongoDB sort document.
//
// Prefixing a field with "-" sorts descending. Use "__" to represent nested fields.
// The special case "id" will be converted to "_id".
//
// Example:
//
//	RequestSort([]string{"-created_at", "user__name"})
//	=>
//	bson.D{
//	  {"created_at", -1},
//	  {"user.name", 1},
//	}
func RequestSort(sort []string) primitive.D {
	result := bson.D{}

	for _, field := range sort {
		order := 1
		if strings.HasPrefix(field, "-") {
			order = -1
			field = strings.TrimPrefix(field, "-")
		}

		// Support field name translation
		if field == "id" {
			field = "_id"
		}

		// Allow dot notation using "__"
		field = strings.ReplaceAll(field, "__", ".")

		result = append(result, bson.E{Key: field, Value: order})
	}

	return result
}

// StructFilter returns a BSON map of specific fields from a struct.
// If no fields are specified, it returns all fields.
func StructFilter(m any, fields ...string) primitive.M {
	raw, _ := bson.Marshal(m)

	var origin map[string]any
	_ = bson.Unmarshal(raw, &origin)

	if len(fields) == 0 {
		return origin
	}

	filtered := make(map[string]any, len(fields))
	for _, f := range fields {
		if val, ok := origin[f]; ok {
			filtered[f] = val
		}
	}

	return filtered
}

func commandMap(c bson.Raw) map[string]any {
	var res map[string]any

	if err := bson.Unmarshal(c, &res); err != nil {
		res = map[string]any{"raw": string(c)}
	}

	return res
}

var commandCache sync.Map

func monitoring() *event.CommandMonitor {
	return &event.CommandMonitor{
		Started: func(ctx context.Context, evt *event.CommandStartedEvent) {
			if evt.CommandName != "ping" && evt.CommandName != "endSessions" {
				commandCache.Store(evt.RequestID, evt.Command)
			}
		},
		Succeeded: func(ctx context.Context, evt *event.CommandSucceededEvent) {
			if evt.CommandName != "ping" && evt.CommandName != "endSessions" {
				if cmd, ok := commandCache.Load(evt.RequestID); ok {
					logger.Info("MGO/CMD SUCCEEDED",
						zap.String("request_id", common.GetContextRequestID(ctx)),
						zap.String("event", evt.CommandName),
						zap.Duration("duration", evt.Duration),
						zap.Any("command", commandMap(cmd.(bson.Raw))),
					)
					commandCache.Delete(evt.RequestID)
				}
			}
		},
		Failed: func(ctx context.Context, evt *event.CommandFailedEvent) {
			if evt.CommandName != "ping" && evt.CommandName != "endSessions" {
				if cmd, ok := commandCache.Load(evt.RequestID); ok {
					logger.Error("MGO/CMD FAILED",
						zap.String("request_id", common.GetContextRequestID(ctx)),
						zap.String("event", evt.CommandName),
						zap.Duration("duration", evt.Duration),
						zap.Any("command", commandMap(cmd.(bson.Raw))),
						zap.Error(errors.New(evt.Failure)),
					)
					commandCache.Delete(evt.RequestID)
				}
			}
		},
	}
}
