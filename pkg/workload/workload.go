package workload


type Generator interface {
	Workload() []string
	Stop() []string

}
