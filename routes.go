package main

func (s *server) routes() {
	s.router.HandleFunc("/", s.handleIndex)
	s.router.HandleFunc("/favicon.ico", s.handleFavicon)
	s.router.HandleFunc("/login", s.postUsernameAndPasswordOnly(s.handleAuthLogin))
	s.router.HandleFunc("/logout", s.loggedInOnly(s.handleAuthLogout))
	s.router.HandleFunc("/register", s.postUsernameAndPasswordOnly(s.handleRegister))
}
