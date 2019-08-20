# Captcha Telegram Bot
A bot to protect the texts or with a captcha.
## Installing
Go to [releases](https://github.com/HirbodBehnam/CaptchaBot/releases) and download one for your operating system. Then head to next part for setting up the bot
### Building From Source
At first get all the required packages:
```bash
go get github.com/dchest/captcha
go get github.com/go-telegram-bot-api/telegram-bot-api
go get github.com/boltdb/bolt
```
Then build the program with
`go build main.go database.go`
## Setting up the bot
Setting up the bot is not very hard and neither very easy. Here is a step to step guide to setup the bot.

At really first you should download or build the bot according to the previous step. Then download the `config.json` file into the same directory as bot.
### Step 1: Getting the Bot Token
At really first, you should create a bot. Contact [BotFather](https://t.me/BotFather) to create a new bot. At the last of the process you should be given a Bot Token. It is like `123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`. Enter it in the `config.json` file after the Token field. (config file example at the next step)
### Step 2: Defining Admins
Admins can add or remove links to database, besides viewing links. So setting a (or more) admin(s) for your bot is mandatory. 

To define admins you need to know the admins ID, and this ID is not the ID that starts with @. It is an int value. To get it you have 2 choices:

1. Use bots on telegram:
You can basically just search the telegram for some ID bots. Here is an example: [myidbot](https://t.me/myidbot)
2. Use this bot: This bot is also capable of returning your own ID yo yourself. Just set the token of your bot in `config.json` and run the bot. Send the bot `/id` and you get your own ID.

After you got your ID, enter it into the `Admins` array in `config.json`. For example if your admins IDs are 1234 and 9876 your config should be like this:
```json
{
  "Token": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
  "Admins": [1234 , 9876],
  "DBName": "database.db"
}
```

After you set the new admins you need to restart the bot.
### Step 3: Running the Bot
After you setup everything, just run the bot.

You can either use a service or just `tmux` to keep the bot alive after you close the SSH connection.
### Step 4: Defining Texts or Links (and Controlling the Bot)
As an admin you can use one of these commands to update the database:
* `/add` : Adds a string or link to database and returns the token to the admin. Users can use the token to access the links or texts.
* `/remove` : Remove a string or text from database by it's token.
* `/cancel` : Cancel removing or adding a text

Also admins can also send a token to bot to access it's data.
