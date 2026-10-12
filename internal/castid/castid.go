// Package castid is how a cast's id follows it: made once, carried between processes in a header, and added to every line logged under it.
package castid

import (
	"context"
	"crypto/rand"
	"log/slog"
)

// header carries the id of the cast a call belongs to.
const header = "Castor-Cast-Id"

type key struct{}

// New is the id of a cast, unguessable since devices fetch what a cast serves by it; the media contract states what it looks like.
func New() string { return rand.Text() }

// With is ctx for the cast id.
func With(ctx context.Context, id string) context.Context { return context.WithValue(ctx, key{}, id) }

// From is the id of the cast ctx belongs to, if any.
func From(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(key{}).(string)
	return id, ok
}

// Tagged is h with every record logged under a cast's context carrying that cast's id.
func Tagged(h slog.Handler) slog.Handler { return tagged{h} }

type tagged struct{ slog.Handler }

func (t tagged) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := From(ctx); ok {
		r.AddAttrs(slog.String("cast_id", id))
	}
	return t.Handler.Handle(ctx, r)
}

func (t tagged) WithAttrs(attrs []slog.Attr) slog.Handler { return tagged{t.Handler.WithAttrs(attrs)} }
func (t tagged) WithGroup(name string) slog.Handler       { return tagged{t.Handler.WithGroup(name)} }
