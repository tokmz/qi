package i18n

// PluralForm 枚举当前支持的复数形式。
// 当前仅支持 zero / one / other，满足英语、中文、日语等大多数语言。
type PluralForm string

const (
	PluralZero  PluralForm = "zero"
	PluralOne   PluralForm = "one"
	PluralOther PluralForm = "other"
)

// PluralFunc 根据数量返回对应 PluralForm。
type PluralFunc func(count int) PluralForm

// SimplePluralFunc 是内置复数规则，适用于英语、中文等语言：
//
//	count == 0 → PluralZero
//	count == 1 → PluralOne
//	others     → PluralOther
func SimplePluralFunc(count int) PluralForm {
	switch count {
	case 0:
		return PluralZero
	case 1:
		return PluralOne
	default:
		return PluralOther
	}
}
