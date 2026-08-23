![linkedin_banner_image_1](https://github.com/Sut103/7DaysPoll-for-Discord/assets/18696845/df4b8411-1915-4d1b-81a2-381c2d8e5324)
# 7DaysPoll-for-Discord

## Overview
7DaysPoll is a Discord Bot that helps you organize events by creating polls with multiple date options. It allows users to vote on their preferred dates, making it easier to find the best time for everyone.

The bot can create polls with 2-7 potential dates starting from a specified date, and it automatically counts unique voters to help you make decisions.

<img width="464" height="704" alt="Example of a native Discord poll created by /poll, with a linked scheduled event" src="https://github.com/user-attachments/assets/c053cf6f-0879-42d2-81fe-e8a66b7191c5" />


## Features
- Create polls with 2-7 consecutive date options
- Customize the starting date
- Add a title to your poll
- Uses Discord's built-in poll feature for easy voting (multiple answers allowed)
- Customizable poll duration (1-32 days)
- Automatically creates a guild scheduled event linked to the poll
- Classic reaction-based voting is still available via `/poll-classic`

## How to use 7DaysPoll on your Discord server

### Adding the Bot to Your Server
Use this invitation link to add the bot to your Discord server:
https://discord.com/api/oauth2/authorize?client_id=1200049972129837107&permissions=580550729485376&scope=bot

### Commands
The bot supports the following slash commands:

- `/poll` - Creates a poll with 7 days starting from today, using Discord's built-in poll feature
- `/poll title:[Your Title]` - Creates a poll with a custom title
- `/poll start-date:MM/DD` - Creates a poll starting from the specified date
- `/poll days:[2-7]` - Creates a poll with the specified number of days (between 2 and 7)
- `/poll duration:[1-32]` - Sets how many days the poll stays open (default is 3)

You can combine these parameters as needed:
- `/poll title:Game Night start-date:05/15 days:5 duration:7` - Creates a poll titled "Game Night" with 5 days starting from May 15th, open for 7 days

The classic embed-based poll with emoji reactions is still available as `/poll-classic`. It supports the same `title`, `start-date`, and `days` parameters and automatically counts unique voters via reactions.

## Development Environment

### Prerequisites
- [Visual Studio Code](https://code.visualstudio.com/)
- [Docker](https://www.docker.com/)
- [VS Code Remote - Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

### Setting Up the Development Environment
1. Clone the repository:
   ```
   git clone https://github.com/Sut103/7dayspoll-for-discord.git
   cd 7dayspoll-for-discord
   ```

2. Create a `.env` file in the root directory with your Discord bot token:
   ```
   DISCORD_BOT_TOKEN=your_discord_bot_token_here
   ```

3. Open the project in VS Code. Once opened, open the Command Palette and select "Remote-Containers: Reopen in Container" to reopen the project inside the container.

4. Once inside the container, you can run the bot:
   ```
   cd app
   go run .
   ```

## Building with Docker

To build the Docker image:

```bash
docker build -t 7dayspoll .
```

This will create a Docker image optimized for arm64 architecture.

## Self-Hosting

### Using Docker

1. Create a `.env` file with your Discord bot token:
   ```
   DISCORD_BOT_TOKEN=your_discord_bot_token_here
   ```

2. Run the Docker container:
   ```bash
   docker run --env-file .env 7dayspoll
   ```

### Manual Deployment

1. Ensure you have Go 1.23 or later installed
2. Clone the repository:
   ```bash
   git clone https://github.com/Sut103/7dayspoll-for-discord.git
   cd 7dayspoll-for-discord/app
   ```

3. Set your Discord bot token:
   ```bash
   export DISCORD_BOT_TOKEN=your_discord_bot_token_here
   ```

4. Run the application:
   ```bash
   go run .
   ```
