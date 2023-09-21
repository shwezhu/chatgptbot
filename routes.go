package main

func (s *server) routes() {
	s.router.HandleFunc("/", s.handleGreeting)
	s.router.HandleFunc("/favicon.ico", s.handleFavicon)
	s.router.HandleFunc("/login", s.postUsernamePasswordOnly(s.handleAuthLogin))
	s.router.HandleFunc("/logout", s.authenticatedOnly(s.handleAuthLogout))
	s.router.HandleFunc("/register", s.postUsernamePasswordOnly(s.handleRegister))
	s.router.HandleFunc("/chat/gpt-3-turbo", s.authenticatedOnly(s.handleGpt3Dot5Turbo))
}
