# Captcha Telegram Bot
A bot to protect the texts or with a captcha or Google reCaptcha.
## Features
* **Nearly Easy Setup**: You can easily setup this bot and use it under 10 minutes (without reCaptcha; Signing up for reCaptcha and registering domain requires more than 30 minutes)
* **reCaptcha Support**: Beside a normal captcha you can use Google's Recaptcha for extra security. reCaptcha V2 and V3 are both supported.
* **Deep Links**: With support of deeplinks, you can instantly send share a link that points to the token. Example: `https://telegram.me/testbot?start=thetoken`; This link will open the bot with the requested token.
* **Small Code Base**: With small code base everyone can study the program.
* **Multi-OS Support**: You can run this bot an _any_ os supported by goLang. You can even run in on Android.
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
## Demos
### Normal Captcha
![Demo Normal](https://media.giphy.com/media/Y3YD8y6kbep9oetbOm/giphy.gif)
### ReCaptcha V2
![Demo V2](https://media.giphy.com/media/h58ZKl4S2xvtxtzoDr/giphy.gif)
### ReCaptcha V3
![Demo V3](https://media.giphy.com/media/gkWoQtiAXOBeckSbpZ/giphy.gif)
## Captcha Modes
### Normal Captcha
Bot sends the user a normal captcha image that contains 8 digits. User must enter the digits via telegram to receive the link.

_Example Image_ ![Example](https://raw.githubusercontent.com/dchest/captcha/master/capgen/example.png)

However these old captchas might not be very secure.
### Recaptcha V2
Bot send the user a link to complete the recaptcha. The site is hosted in the bot's server. **A domain name is required for the bot to work**.
### Recaptcha V3
Recaptcha V3 is different from all of the other captchas. It does not require the user to do anything; Instead, it rates the user from 0.0 to 1.0. The more this number is, the user is more likely a human.

So how this bot works? At first you should define a minimum pass score for the users that request the captcha. At first the bot gets the user's score. It checks it with the minimum score, if it's less than it, _bot refuses to send the user the link or text_. So method is very strict. If the user does not achieve the minimum score, there is no way for user to get the link. On client side, then the user sends the token to bot, bot send them a link to the server. After the user opens it, he must just wait for the captcha to give the user a score. Bot automatically sends them the link or text afterwards.

If the user receives a low score this will be shown to him: `Unfortunately you are not worthy enough to access this right now.`
## Setting up the bot
At really first you should download or build the bot according to the previous step. Then download the `config.json` file into the same directory as bot.
### Getting Bot Token
At really first, you should create a bot. Contact [BotFather](https://t.me/BotFather) to create a new bot. At the last of the process you should be given a Bot Token. It is like `123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`. Enter it in the `config.json` file after the Token field. (config file example at the next step)
### Defining Admins
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
### Defining The Captcha Mode
As described above there are 2 different captcha modes
#### Normal Captcha
You do not need to do anything more. Just move to next step
#### Recaptcha
At first, thanks to [this](https://github.com/dpapathanasiou/go-recaptcha) project I implemented a custom recaptcha API in my own app.

Before configuring the bot you should do three things:
* Set a domain on your server
* Get the reCaptcha tokens
* Open a port for recaptcha webserver on your server
##### Domain
Domain is required in order to recaptcha work. You can use a free domain at [Now-DNS](https://now-dns.com/) and register the _domain_ (not the sub-domain) at the google admin console.
##### reCaptcha Tokens
reCaptcha Works with 2 tokens: _Private Key_ and _Site Key_. To generate one and register go to [here](https://www.google.com/recaptcha/admin) and register. While registering, you will be asked to choose between reCaptcha V2 and V3; If you wish to create V3, just choose the radio button and continue; but If you want to choose the V2, make sure you choose `"I'm not a robot" Checkbox` radio button; You will be given a site key and private key. You need them for the config file.
##### Opening Firewall
The bot will listen for http connections on a port. That port must be opened in your firewall.
##### Configuring Bot
To make the bot work with reCaptcha you should add a Recaptcha object to config.json. Example for V2 reCaptcha:
```json
{
  "Token": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
  "Admins": [1234],
  "DBName": "database.db",
  "Recaptcha": {
    "V2" : true,
    "PrivateKey" : "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "PublicKey": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "Domain" : "demo.test.com",
    "Port" : 8080
  }
}
```
Here:
* `V2: true` tells the bot that is should use reCaptcha V2
* `Private Key` is the secret key that reCaptcha admin panel gave you
* `Public Key` is the site key that reCaptcha admin panel gave you
* `Domain` is the domain that points to your server IP
* `Port` is the port that bot starts the webserver on it; This should not be in use

Example for V3:
```json
{
  "Token": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
  "Admins": [1234],
  "DBName": "database.db",
  "Recaptcha": {
    "V2" : false,
    "PrivateKey" : "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "PublicKey": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "Domain" : "demo.test.com",
    "MinScore": 0.6,
    "Port" : 8080
  }
}
```
Here:
* `V2: false` tells the bot that is should use reCaptcha V3
* `Private Key` is the secret key that reCaptcha admin panel gave you
* `Public Key` is the site key that reCaptcha admin panel gave you
* `Domain` is the domain that points to your server IP
* `MinScore` is the minimum score that user requires to get the link. Should be between 0 and 1. A reasonable value is 0.5 or 0.6
* `Port` is the port that bot starts the webserver on it; This should not be in use
### Running the Bot
After you setup everything, just run the bot.

You can either use a service or just `tmux` to keep the bot alive after you close the SSH connection.
### Defining Texts or Links (and Controlling the Bot)
As an admin you can use one of these commands to update the database:
* `/add` : Adds a string or link to database and returns the token to the admin. Users can use the token to access the links or texts.
* `/remove` : Remove a string or text from database by it's token.
* `/cancel` : Cancel removing or adding a text
* `/list` : Lists all of the keys and values in database

Admins can also send a token to bot to access it's data.
