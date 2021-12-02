package domain

type Action int

const (
	ActionSell = Action(iota)
	ActionBuy
	ActionNothing
)
