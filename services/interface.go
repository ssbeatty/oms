package services

type Service interface {
	Start()
	Close()
}
