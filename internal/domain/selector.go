package domain

// Strategy pattern: pick one forwarder from the alive set.

type ForwarderSelector interface {
	Select([]Forwarder) *Forwarder
	Name() string
}

type RoundRobinSelector struct {
	idx int
}

func (r *RoundRobinSelector) Name() string { return "round_robin" }

func (r *RoundRobinSelector) Select(f []Forwarder) *Forwarder {
	if len(f) == 0 {
		return nil
	}
	r.idx = (r.idx + 1) % len(f)
	return &f[r.idx]
}
