package services

type PrintService interface {
	Print(a ...interface{})
	Printf(format string, a ...interface{})
}
