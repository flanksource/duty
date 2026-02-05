package db

import (
	"fmt"

	"github.com/samber/oops"
	"gorm.io/gorm"
)

// oops plugin simply wraps any gorm error
// with oops and adds the "db" tag to it.
type oopsPlugin struct {
}

func NewOopsPlugin() gorm.Plugin {
	return &oopsPlugin{}
}

func (p oopsPlugin) Name() string {
	return "flanksource-oops"
}

type gormHookFunc func(tx *gorm.DB)

type gormRegister interface {
	Register(name string, fn func(*gorm.DB)) error
}

func (p oopsPlugin) Initialize(db *gorm.DB) (err error) {
	cb := db.Callback()
	hooks := []struct {
		callback gormRegister
		hook     gormHookFunc
		name     string
	}{
		{cb.Create().After("flanksource-oops:create"), p.after(), "after:create"},
		{cb.Query().After("flanksource-oops:query"), p.after(), "after:select"},
		{cb.Delete().After("flanksource-oops:delete"), p.after(), "after:delete"},
		{cb.Update().After("flanksource-oops:update"), p.after(), "after:update"},
		{cb.Row().After("flanksource-oops:row"), p.after(), "after:row"},
		{cb.Raw().After("flanksource-oops:raw"), p.after(), "after:raw"},
	}

	var firstErr error
	for _, h := range hooks {
		if err := h.callback.Register("oops:"+h.name, h.hook); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("callback register %s failed: %w", h.name, err)
		}
	}

	return firstErr
}

func (p *oopsPlugin) after() gormHookFunc {
	return func(tx *gorm.DB) {
		if tx.Error != nil {
			tx.Error = oops.Tags("db").Wrap(ErrorDetails(tx.Error))
		}
	}
}
