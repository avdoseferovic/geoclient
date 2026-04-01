package net

// Sequencer tracks packet sequence numbers for the EO protocol.
type Sequencer struct {
	start   int
	counter int
}

func NewSequencer() *Sequencer {
	return &Sequencer{start: 0, counter: 0}
}

// NextSequence returns the next sequence value and advances the counter (0-9 cycle).
func (s *Sequencer) NextSequence() int {
	result := s.start + s.counter
	s.counter = (s.counter + 1) % 10
	return result
}

// SetStart sets the sequence start value without resetting the counter.
func (s *Sequencer) SetStart(start int) {
	s.start = start
}

// Reset sets the sequence start and resets the counter to 0.
func (s *Sequencer) Reset(start int) {
	s.start = start
	s.counter = 0
}
