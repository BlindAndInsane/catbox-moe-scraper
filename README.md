# Catbox.moe Scraper
simple scraper that checks random links for catbox file host and if it finds a file it downloads it and save its link to db.

## How to use
- Clone the repo
- Install go
- run `go build`
- rename config-example.yaml to config.yaml
- change the config to whatever u want
- run the binary

## Commands
i have added few commands for QOL.<br>
- stop (stops the program gracefully)
- pause (pauses the program)
- resume (resumes the program)

just type these when code is running and press enter and it will work no fancy tui is made so ya just use whatever u have in hand.

or just press ctrl+c to stop it gracefully as i have made it recieve that signal and but app in stop state