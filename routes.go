package main

func (s *server) routes() {
	s.router.HandleFunc("/", s.handleIndex)
	s.router.HandleFunc("/favicon.ico", s.handleFavicon)
	s.router.HandleFunc("/login", s.handleAuthLogin)
	s.router.HandleFunc("/login", s.handleAuthLogout)
}
