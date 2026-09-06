// Package builders provides fluent constructors for model.* types so a
// test only has to set the field it's actually testing, not every
// required-looking field a literal struct expression would otherwise force
// it to name.
package builders

import "menata.id/app/internal/model"

// MachineBuilder builds a *model.Machine.
type MachineBuilder struct {
	m *model.Machine
}

// Machine starts a MachineBuilder with id, a matching Name, and a fixed
// test ApplicationID -- override either with the setters below if a test
// cares.
func Machine(id string) *MachineBuilder {
	return &MachineBuilder{m: &model.Machine{ID: id, ApplicationID: "app_test", Name: id}}
}

func (b *MachineBuilder) Name(name string) *MachineBuilder {
	b.m.Name = name
	return b
}

func (b *MachineBuilder) ApplicationID(id string) *MachineBuilder {
	b.m.ApplicationID = id
	return b
}

// WithField appends f, stamping f.MachineID to this machine's ID.
func (b *MachineBuilder) WithField(f *model.Field) *MachineBuilder {
	f.MachineID = b.m.ID
	b.m.Fields = append(b.m.Fields, f)
	return b
}

// WithEvent appends e, stamping e.MachineID to this machine's ID.
func (b *MachineBuilder) WithEvent(e *model.Event) *MachineBuilder {
	e.MachineID = b.m.ID
	b.m.Events = append(b.m.Events, e)
	return b
}

func (b *MachineBuilder) Build() *model.Machine {
	return b.m
}

// FieldBuilder builds a *model.Field.
type FieldBuilder struct {
	f *model.Field
}

// Field starts a FieldBuilder with id, a matching Name, and t.
func Field(id string, t model.FieldType) *FieldBuilder {
	return &FieldBuilder{f: &model.Field{ID: id, Name: id, Type: t}}
}

func (b *FieldBuilder) Name(name string) *FieldBuilder {
	b.f.Name = name
	return b
}

func (b *FieldBuilder) Required() *FieldBuilder {
	b.f.Required = true
	return b
}

func (b *FieldBuilder) Position(p int) *FieldBuilder {
	b.f.Position = p
	return b
}

func (b *FieldBuilder) Options(o model.FieldOptions) *FieldBuilder {
	b.f.Options = o
	return b
}

func (b *FieldBuilder) Build() *model.Field {
	return b.f
}

// EventBuilder builds a *model.Event.
type EventBuilder struct {
	e *model.Event
}

// Event starts an EventBuilder with id and name.
func Event(id, name string) *EventBuilder {
	return &EventBuilder{e: &model.Event{ID: id, Name: name}}
}

func (b *EventBuilder) WithAction(a *model.EventAction) *EventBuilder {
	b.e.Actions = append(b.e.Actions, a)
	return b
}

func (b *EventBuilder) Condition(c *model.ConstraintExpression) *EventBuilder {
	b.e.Condition = c
	return b
}

func (b *EventBuilder) Build() *model.Event {
	return b.e
}
