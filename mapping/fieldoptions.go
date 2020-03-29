package mapping

import "fmt"

type (
	// use context and OptionalDep option to determine the value of Optional
	// nothing to do with context.Context
	fieldOptionsWithContext struct {
		FromString bool
		Optional   bool
		Options    []string
		Default    string
		Range      *numberRange
	}

	fieldOptions struct {
		fieldOptionsWithContext
		OptionalDep string
	}

	numberRange struct {
		left         float64
		leftInclude  bool
		right        float64
		rightInclude bool
	}
)

func (o *fieldOptionsWithContext) fromString() bool {
	return o != nil && o.FromString
}

func (o *fieldOptionsWithContext) getDefault() (string, bool) {
	if o == nil {
		return "", false
	} else {
		return o.Default, len(o.Default) > 0
	}
}

func (o *fieldOptionsWithContext) optional() bool {
	return o != nil && o.Optional
}

func (o *fieldOptionsWithContext) options() []string {
	if o == nil {
		return nil
	}

	return o.Options
}

func (o *fieldOptions) optionalDep() string {
	if o == nil {
		return ""
	} else {
		return o.OptionalDep
	}
}

func (o *fieldOptions) toOptionsWithContext(key string, m Valuer) (*fieldOptionsWithContext, error) {
	var optional bool
	if o.optional() {
		dep := o.optionalDep()
		if len(dep) == 0 {
			optional = true
		} else if _, ok := m.Value(dep); ok {
			optional = false
		} else {
			// the dependant not provided, so the value for field cannot be provided
			if _, ok := m.Value(key); ok {
				return nil, fmt.Errorf("value provided for %s, but not for %s", key, dep)
			} else {
				// because not provided, we just set it to optional, whose value will never be set afterwards.
				optional = true
			}
		}
	}

	if o.fieldOptionsWithContext.Optional == optional {
		return &o.fieldOptionsWithContext, nil
	} else {
		return &fieldOptionsWithContext{
			FromString: o.FromString,
			Optional:   optional,
			Options:    o.Options,
			Default:    o.Default,
		}, nil
	}
}
