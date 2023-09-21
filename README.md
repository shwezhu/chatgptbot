# gptbot

Use Sqlite3 store username, password, and token balance.

Use [gorilla/session](https://github.com/gorilla/sessions) for session management, and save session in [Redis](https://redis.io/docs/getting-started/installation/).

## usage

When request sent with browser, you don't need to specify session_id which stored in cookies already, when send a http request, your browser will carry cookie for you automatically.

```shell
# register
$ curl localhost:8080/register -d "username=david&password=778899a" -v
# login
$ curl localhost:8080/login -d "username=david&password=778899a" -v
# logout
$ curl localhost:8080/logout --cookie "session_id=your_session_id" -v
# chat with chat gpt
$ curl localhost:8080/chat/gpt-turbo -d "message=what's your name" --cookie "session_id=your_session_id"
```

## issues

- Just use cookie is not safe, if someone gets the cookie of a user he/her may send authenticated request
- User that has logged in, but send request without cookie, the server will create a new session for this connection
- Encoding messages in session can use gob for better performance
- 