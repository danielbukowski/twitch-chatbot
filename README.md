# Twitch Bot

This project is a chatbot for Twitch.tv. It listens for messages on a channel and do something.  
For example, the most basic command is a Ping command. It responds with a message 'Pong' whoever types the message `!ping` in a chat.

I've created this project with an idea in mind to learn Go and rewrite an older project made in TypeScript.



## Environment Variables

All required environment variables to this project are all listed in the config package.



## How To Run 

At the time writing this readme, I build this project using GCC, Make and Go 1.22.5.

To run database migration scripts you need to download [Goose](https://github.com/pressly/goose) on your local machine.

In the near future I will make the building process easier by using Docker/Podman.



## License

[MIT](https://github.com/danielbukowski/twitch-chatbot/blob/main/LICENSE)
