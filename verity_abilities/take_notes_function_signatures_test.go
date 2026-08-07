package verity_abilities_test

import (
	verity "github.com/verity-bdd/verity-bdd"
	"github.com/verity-bdd/verity-bdd/verity_abilities/take_notes"
)

var (
	_ func() verity.Ability                             = take_notes.UsingEmptyNotepad
	_ func(*take_notes.NoteBook) verity.Ability         = take_notes.Using
	_ func() *take_notes.NoteBook                       = take_notes.NewNoteBook
	_ func(map[string]any) *take_notes.NoteBook         = take_notes.NotepadWith
	_ func(any) interface{ As(string) verity.Activity } = take_notes.TakeNoteOf
	_ func(string) verity.Question[any]                 = take_notes.NoteValue
)
