# gptbot

## issues

- Just use cookie is not safe, cause anyone get the cookie, can send authenticated request
- User that has logged in, but send request without cookie, the server will create a new session for this connection
- Encoding messages in session can use gob for better performance