package i18n

// en is the English catalog and the source of truth: it must contain every key
// used by the bot. Other languages fall back to these strings.
var en = map[string]string{
	// Footers ---------------------------------------------------------------
	"footer.hint":            "💡 Hint: Use /help to list the available commands.",
	"footer.selection_saved": "Selections are saved on every change.",

	// /report ---------------------------------------------------------------
	"report.title":           "Report",
	"report.user_not_found":  "The specified user could not be found. Please check the ID and try again.",
	"report.no_mod_roles":    "No moderation role is configured for this server. Please contact an administrator.",
	"report.error_generic":   "An error occurred while processing your request. Please contact a moderator.",
	"report.no_mod_channels": "No moderation channel is configured for this server. Please contact an administrator.",
	"report.description":     "**Reported user**: <@!%s> (%s | %s)\n**Channel**: <#%s>\n**Reason**: %s\n\n**Recent messages**:\n%s",
	"report.alert_title":     "🚨 New report by %s",
	"report.alert_footer":    "No action has been taken yet.",
	"report.received":        "User %s has been reported to the moderation team.",
	"report.received_footer": "Thank you for helping keep this server healthy.",
	"report.no_reason":       "No reason provided",

	// /help -----------------------------------------------------------------
	"help.title":          "💡 Help",
	"help.description":    "Here is the list of available commands:",
	"help.options_label":  "**Options**:",
	"help.required_label": "**Required:**",
	"help.optional_label": "**Optional:**",

	// Automod (DM warning + escalation) -------------------------------------
	"automod.warning":               "⚠️ **Warning %d/%d**\nYour message on the server [%s] was deleted because %s\n\nHere is a copy of your message:\n",
	"automod.cause_banned_word":     "it contains the word `%s`",
	"automod.cause_banned_website":  "it contains a forbidden link: `%s`",
	"automod.reason_banned_word":    "The word `%s` is banned",
	"automod.reason_banned_website": "The website `%s` is banned",
	"automod.violation_suffix":      "%s (violation %d/%d)",
	"automod.ban_reason":            "%s\n%d automod violations",
	"automod.spam_reason":           "Spam (message repeated %d times across %d channels in %s)",

	// Automod alert (sent to log channels) ----------------------------------
	"automod.alert.title":       "🚨 Automoderation action triggered",
	"automod.alert.description": "**User**: <@%s> (%s | %s)\n**Action**: `%s`\n**Triggered rule**: `%s`\n**Reason**: %s\n\n**Recent messages**: \n%s",

	// Moderation actions (kick/ban DMs) -------------------------------------
	"action.kick_dm": "👢 **Kick**\n\nYou have been kicked from the server [%s] for the following reason: \n*%s*",
	"action.ban_dm":  "🔨 **Ban**\n\nYou have been banned from the server [%s] for the following reason: \n*%s*",

	// Recent-message rendering ----------------------------------------------
	"msg.files_embeds":   "<file(s): %d + embed(s)>",
	"msg.file":           "<file>",
	"msg.files":          "<%d files>",
	"msg.embed":          "<embed>",
	"msg.sticker":        "<sticker>",
	"msg.stickers":       "<%d stickers>",
	"msg.empty":          "<empty message>",
	"msg.attachment_one": " [+file]",
	"msg.attachments":    " [+%d files]",
	"msg.more":           "... and %d more message(s)",
	"msg.none_recent":    "*No recent message available*",

	// Banned words ----------------------------------------------------------
	"bannedword.title":            "Banned word",
	"bannedword.empty_word":       "The word cannot be empty.",
	"bannedword.add_error":        "An error occurred while adding the banned word.",
	"bannedword.type_literal":     "literal",
	"bannedword.type_regex":       "regex",
	"bannedword.added_title":      "✅ Banned word added",
	"bannedword.added":            "The word `%s` (type: %s) has been added to the banned words list.",
	"bannedword.list_error_title": "Banned words",
	"bannedword.list_error":       "An error occurred while fetching the banned words.",
	"bannedword.list_empty_title": "📝 Banned words",
	"bannedword.list_empty":       "No banned word is configured for this server.",
	"bannedword.list_item":        "• **ID %d**: `%s` (type: %s)\n",
	"bannedword.list_title":       "📝 Banned words list",
	"bannedword.list_total":       "**Total**: %d banned word(s)\n\n%s",

	// Banned websites -------------------------------------------------------
	"bannedsite.title":            "Banned website",
	"bannedsite.empty_url":        "The URL cannot be empty.",
	"bannedsite.add_error":        "An error occurred while adding the banned website.",
	"bannedsite.added_title":      "✅ Banned website added",
	"bannedsite.added":            "The website `%s` has been added to the banned websites list.",
	"bannedsite.list_error_title": "Banned websites",
	"bannedsite.list_error":       "An error occurred while fetching the banned websites.",
	"bannedsite.list_empty_title": "🌐 Banned websites",
	"bannedsite.list_empty":       "No banned website is configured for this server.",
	"bannedsite.list_item":        "• **ID %d**: `%s`\n",
	"bannedsite.list_title":       "🌐 Banned websites list",
	"bannedsite.list_total":       "**Total**: %d banned website(s)\n\n%s",

	// Configure automod -----------------------------------------------------
	"automodcfg.error_title": "Automoderation configuration",
	"automodcfg.error":       "An error occurred while fetching the features.",

	// /status ---------------------------------------------------------------
	"status.title":         "Status",
	"status.uptime":        "Uptime",
	"status.servers":       "Connected servers",
	"status.servers_value": "%d servers:\n%s",
	"status.cpu":           "CPU usage",
	"status.memory":        "Memory usage",
	"status.goroutines":    "Goroutines",
	"status.rss":           "RSS memory",
	"status.log_entries":   "Log entries",
	"status.last_errors":   "Last 5 errors",

	// /get-message-history --------------------------------------------------
	"history.title":       "Message History",
	"history.limit_error": "The limit must be between 1 and 1000.",
	"history.empty":       "No message in cache for this server.",
	"history.header":      "Last %d cached messages:\n\n",

	// /get-bot-logs and /get-moderation-logs --------------------------------
	"logs.bot_title": "Bot logs",
	"logs.bot_error": "Error while fetching the bot logs.",
	"logs.bot_empty": "No log found.",
	"logs.mod_title": "Moderation logs",
	"logs.mod_error": "Error while fetching the moderation logs.",
	"logs.mod_empty": "No moderation log found.",
	"logs.latest":    "**Latest logs** (%d/%d)\n\n%s",

	// Selection menus (channels / roles / automod) --------------------------
	"selector.error_channels_title": "Channel selection",
	"selector.error_channels":       "An error occurred while fetching the channels.",
	"selector.error_roles_title":    "Role selection",
	"selector.error_roles":          "An error occurred while fetching the roles.",
	"selector.no_channel":           "⚠️ No channel selected",
	"selector.no_role":              "⚠️ No role selected.",
	"selector.items_selected":       "**%d**/**%d** item(s) selected.",
	"selector.updating":             "Updating...",
	"selector.config_done_title":    "Configuration complete",

	"selector.log.title":       "Log channels selection",
	"selector.log.description": "Select the log channels then click \"Done\".\nThese are the channels where moderation actions will be logged.",
	"selector.log.placeholder": "Select the log channels",
	"selector.log.done_header": "✅ Log channels selected:",

	"selector.mod.title":       "Moderation channels selection",
	"selector.mod.description": "Select the moderation channels then click \"Done\".\nThese are the channels where moderators will be notified.",
	"selector.mod.placeholder": "Select the moderation channels",
	"selector.mod.done_header": "✅ Moderation channels selected:",

	"selector.excluded.title":       "Excluded channels selection",
	"selector.excluded.description": "Select the excluded channels then click \"Done\".\nThese are the channels that will be ignored by automoderation.",
	"selector.excluded.placeholder": "Select the excluded channels",
	"selector.excluded.done_header": "✅ Excluded channels selected:",

	"selector.role.title":       "Moderation roles selection",
	"selector.role.description": "Select the moderation roles then click \"Done\".\nThese are the roles that administer the server and will be notified.\n\n⚠️ Warning: if you don't hold at least one of these roles, you will no longer be able to use the administration commands!",
	"selector.role.placeholder": "Select the moderation roles",
	"selector.role.done":        "✅ %d roles selected:\n%s",

	"selector.automod.title":       "⚙️ Automoderation configuration",
	"selector.automod.description": "Select the automoderation features to enable then click \"Done\".",
	"selector.automod.placeholder": "Select the features to enable",
	"selector.automod.none":        "⚠️ All automoderation features are disabled",
	"selector.automod.done":        "✅ Configuration saved successfully!\n\n**Enabled features** (%d):\n%s",

	"automod_feature.banned_words.name":    "📝 Banned words",
	"automod_feature.banned_words.desc":    "Detects and deletes messages containing banned words",
	"automod_feature.banned_websites.name": "🌐 Banned websites",
	"automod_feature.banned_websites.desc": "Blocks links to banned websites",
	"automod_feature.spam.name":            "🚨 Spam detection",
	"automod_feature.spam.desc":            "Automatically bans users who spam messages",

	// Navigation buttons (shared by selectors) ------------------------------
	"button.prev": "◀️ Previous",
	"button.next": "Next ▶️",
	"button.page": "Page %d/%d",
	"button.done": "✅ Done",

	// Removal menus (banned word / website) ---------------------------------
	"remove.total":       "**Total**: %d entry/entries. **Page %d/%d**.",
	"remove.notice_none": "⚠️ No entry was removed.",
	"remove.notice":      "✅ %d %s removed: %s",
	"remove.type":        "type: %s",

	"remove.word.title":       "🗑️ Remove a banned word",
	"remove.word.intro":       "Select the word(s) to remove from the banned words list.",
	"remove.word.placeholder": "Select the words to remove",
	"remove.word.empty":       "No banned word is configured for this server.",
	"remove.word.error_title": "Banned words",
	"remove.word.error":       "An error occurred while fetching the banned words.",
	"remove.word.noun":        "word(s)",

	"remove.site.title":       "🗑️ Remove a banned website",
	"remove.site.intro":       "Select the website(s) to remove from the banned websites list.",
	"remove.site.placeholder": "Select the websites to remove",
	"remove.site.empty":       "No banned website is configured for this server.",
	"remove.site.error_title": "Banned websites",
	"remove.site.error":       "An error occurred while fetching the banned websites.",
	"remove.site.noun":        "website(s)",

	// Report action buttons (kick / ban / ignore) ---------------------------
	"report_action.error_title":      "Error",
	"report_action.fetch_user_error": "Unable to fetch the user's information.",
	"report_action.kick_dm":          "👢 **Kick**\n\nYou have been kicked from the server [%s] on <t:%d:f> for the following reason: \n`Kicked following a report`",
	"report_action.ban_dm":           "🔨 **Ban**\n\nYou have been banned from the server [%s] on <t:%d:f> for the following reason: \n`Banned following a report`",
	"report_action.kicked":           "✅ User **%s** has been kicked from the server.",
	"report_action.banned":           "✅ User **%s** has been banned from the server.",
	"report_action.ignored":          "✅ The report for user **%s** has been ignored.",
	"report_action.action_error":     "Unable to perform the action. Make sure the bot has the required permissions and that the user is still on the server.",
	"report_action.done_title":       "Action performed",
	"report_action.taken":            "\n\nAction '%s' performed by %s on <t:%d:f>",
	"report_action.audit_kick":       "Kicked on %s following a report",
	"report_action.audit_ban":        "Banned on %s following a report",

	// /lang -----------------------------------------------------------------
	"lang.menu_title":       "🌐 Server language",
	"lang.menu_description": "Select the bot language for this server.",
	"lang.placeholder":      "Select a language",
	"lang.updated":          "✅ The server language is now **%s**.",
	"lang.error":            "An error occurred while changing the language.",

	// Slash-command descriptions (shown in the command picker) --------------
	"cmd.report.desc":                   "Report a user to the moderation team",
	"cmd.report.opt.user":               "The user to report",
	"cmd.report.opt.reason":             "The reason for the report",
	"cmd.status.desc":                   "Show the bot status",
	"cmd.get-message-history.desc":      "Show the guild's cached message history",
	"cmd.get-message-history.opt.limit": "Number of messages to fetch (default 10, max 1000)",
	"cmd.get-bot-logs.desc":             "Show the latest bot logs",
	"cmd.get-moderation-logs.desc":      "Show the latest moderation logs",
	"cmd.set-log-channels.desc":         "Select the channels where moderation reports will be sent",
	"cmd.set-moderation-channels.desc":  "Select the channels where reports will be sent",
	"cmd.set-excluded-channels.desc":    "Select the channels excluded from automoderation",
	"cmd.set-moderation-roles.desc":     "Select the roles that will have moderation permissions",
	"cmd.add-banned-word.desc":          "Add a word or expression to the banned words list",
	"cmd.add-banned-word.opt.word":      "The word or regex to ban",
	"cmd.add-banned-word.opt.is_regex":  "Whether the word is a regex",
	"cmd.get-banned-words.desc":         "Show the banned words list",
	"cmd.remove-banned-word.desc":       "Remove one or more words from the banned words list",
	"cmd.add-banned-website.desc":       "Add a website to the banned websites list",
	"cmd.add-banned-website.opt.url":    "The URL of the website to ban (e.g. example.com)",
	"cmd.get-banned-websites.desc":      "Show the banned websites list",
	"cmd.remove-banned-website.desc":    "Remove one or more websites from the banned websites list",
	"cmd.configure-automod.desc":        "Enable or disable automoderation features",
	"cmd.help.desc":                     "List all available commands",
	"cmd.lang.desc":                     "Set the bot language for this server",
}
