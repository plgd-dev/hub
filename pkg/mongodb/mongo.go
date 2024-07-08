package mongodb

type Operator = string

const (
	And    Operator = "$and"
	Or     Operator = "$or"
	Exists Operator = "$exists"
	All    Operator = "$all"
	In     Operator = "$in"
	Set    Operator = "$set"
	Unset  Operator = "$unset"
	Match  Operator = "$match"
)
