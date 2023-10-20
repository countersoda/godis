package app

func NewGodis(addr string) {
	server := NewServer(addr)
	server.Run()
}
