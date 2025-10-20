# discord-bot-moderation
A Discord bot to auto moderate a server because nobody want spams

# Installation
1. Go to https://discord.com/developers/applications and create a new application.
2. Go to the "Installation" tab and set "Install link" to None.
3. Navigate to the "Bot" tab and click "Reset Token" to generate a new bot token. Copy this token and add it to the `.env` file as `DISCORD_BOT_TOKEN=your_bot_token_here`.
4. Uncheck the "Public Bot" option.
5. Enable "Presence Intent", "Server Members Intent", and "Message Content Intent" under the "Privileged Gateway Intents" section.
6. Copy the ID of your application from the "General Information" tab.
7. Go to the following URL in your browser, replacing `YOUR_CLIENT_ID` with your application's client ID:
   ```https://discord.com/oauth2/authorize?client_id=YOUR_CLIENT_ID&permissions=274877926400&scope=bot%20applications.commands```

# Usage
1. When the bot is added to a server, it will automatically start moderating messages based on predefined rules.
2. You need to use the `/set-moderation-channels` command to specify which channels the bot should monitor for moderation.
3. You also need to use the `/set-log-channel` command to specify which channel the bot should use to log moderation actions.
4. And the `/set-admin-roles` command to specify which role has permission to manage the bot settings and be notified of moderation actions.

// TODO : Replace this with `/setup` command that does all the setup in one go.
