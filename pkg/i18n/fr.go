package i18n

// fr is the French catalog. These are the bot's original hardcoded strings.
var fr = map[string]string{
	// Footers ---------------------------------------------------------------
	"footer.hint":            "💡 Hint: Utilisez /help pour lister les commandes disponibles.",
	"footer.selection_saved": "Les sélections sont sauvegardées à chaque modification",

	// /report ---------------------------------------------------------------
	"report.title":           "Signalement",
	"report.user_not_found":  "L'utilisateur spécifié est introuvable. Veuillez vérifier l'ID et réessayer.",
	"report.no_mod_roles":    "Aucun rôle de modération n'est configuré pour ce serveur. Veuillez contacter un administrateur.",
	"report.error_generic":   "Une erreur est survenue lors du traitement de votre demande. Contactez un modérateur.",
	"report.no_mod_channels": "Aucun canal de modération n'est configuré pour ce serveur. Veuillez contacter un administrateur.",
	"report.description":     "**Utilisateur signalé**: <@!%s> (%s | %s)\n**Salon**: <#%s>\n**Raison**: %s\n\n**Messages récents**:\n%s",
	"report.alert_title":     "🚨 Nouveau signalement par %s",
	"report.alert_footer":    "Aucune action n'a encore été prise.",
	"report.received":        "L'utilisateur %s a été signalé à la moderation",
	"report.received_footer": "Merci de rendre ce serveur plus sain.",
	"report.no_reason":       "Aucune raison fournie",

	// /help -----------------------------------------------------------------
	"help.title":          "💡 Help",
	"help.description":    "Voici la liste des commandes disponibles :",
	"help.options_label":  "**Options**:",
	"help.required_label": "**Requis :**",
	"help.optional_label": "**Optionel :**",

	// Automod (DM warning + escalation) -------------------------------------
	"automod.warning":               "⚠️ **Avertissement %d/%d**\nVotre message sur le serveur [%s] a été supprimé car %s\n\nVoici une copie de votre message :\n",
	"automod.cause_banned_word":     "il contient le mot `%s`",
	"automod.cause_banned_website":  "il contient un lien interdit : `%s`",
	"automod.reason_banned_word":    "Le mot `%s` est banni",
	"automod.reason_banned_website": "Le site `%s` est banni",
	"automod.violation_suffix":      "%s (violation %d/%d)",
	"automod.ban_reason":            "%s\n%d violations d'automodération",
	"automod.spam_reason":           "Spam (message répété %d fois dans %d salons en %s)",

	// Automod alert (sent to log channels) ----------------------------------
	"automod.alert.title":       "🚨 Action d'automodération déclenchée",
	"automod.alert.description": "**Utilisateur**: <@%s> (%s | %s)\n**Action**: `%s`\n**Triggered rule**: `%s`\n**Raison**: %s\n\n**Messages récents**: \n%s",

	// Moderation actions (kick/ban DMs) -------------------------------------
	"action.kick_dm": "👢 **Expulsion**\n\nVous avez été expulsé du serveur [%s] pour la raison suivante : \n*%s*",
	"action.ban_dm":  "🔨 **Bannissement**\n\nVous avez été banni du serveur [%s] pour la raison suivante : \n*%s*",

	// Recent-message rendering ----------------------------------------------
	"msg.files_embeds":   "<fichier(s): %d + embed(s)>",
	"msg.file":           "<fichier>",
	"msg.files":          "<%d fichiers>",
	"msg.embed":          "<embed>",
	"msg.sticker":        "<sticker>",
	"msg.stickers":       "<%d stickers>",
	"msg.empty":          "<message vide>",
	"msg.attachment_one": " [+fichier]",
	"msg.attachments":    " [+%d fichiers]",
	"msg.more":           "... et %d message(s) de plus",
	"msg.none_recent":    "*Aucun message récent disponible*",

	// Banned words ----------------------------------------------------------
	"bannedword.title":            "Mot interdit",
	"bannedword.empty_word":       "Le mot ne peut pas être vide.",
	"bannedword.add_error":        "Une erreur est survenue lors de l'ajout du mot interdit.",
	"bannedword.type_literal":     "littéral",
	"bannedword.type_regex":       "regex",
	"bannedword.added_title":      "✅ Mot interdit ajouté",
	"bannedword.added":            "Le mot `%s` (type: %s) a été ajouté à la liste des mots interdits.",
	"bannedword.list_error_title": "Mots interdits",
	"bannedword.list_error":       "Une erreur est survenue lors de la récupération des mots interdits.",
	"bannedword.list_empty_title": "📝 Mots interdits",
	"bannedword.list_empty":       "Aucun mot interdit n'est configuré pour ce serveur.",
	"bannedword.list_item":        "• **ID %d**: `%s` (type: %s)\n",
	"bannedword.list_title":       "📝 Liste des mots interdits",
	"bannedword.list_total":       "**Total**: %d mot(s) interdit(s)\n\n%s",

	// Banned websites -------------------------------------------------------
	"bannedsite.title":            "Site web interdit",
	"bannedsite.empty_url":        "L'URL ne peut pas être vide.",
	"bannedsite.add_error":        "Une erreur est survenue lors de l'ajout du site web interdit.",
	"bannedsite.added_title":      "✅ Site web interdit ajouté",
	"bannedsite.added":            "Le site `%s` a été ajouté à la liste des sites web interdits.",
	"bannedsite.list_error_title": "Sites web interdits",
	"bannedsite.list_error":       "Une erreur est survenue lors de la récupération des sites web interdits.",
	"bannedsite.list_empty_title": "🌐 Sites web interdits",
	"bannedsite.list_empty":       "Aucun site web interdit n'est configuré pour ce serveur.",
	"bannedsite.list_item":        "• **ID %d**: `%s`\n",
	"bannedsite.list_title":       "🌐 Liste des sites web interdits",
	"bannedsite.list_total":       "**Total**: %d site(s) web interdit(s)\n\n%s",

	// Configure automod -----------------------------------------------------
	"automodcfg.error_title": "Configuration Automodération",
	"automodcfg.error":       "Une erreur est survenue lors de la récupération des fonctionnalités.",

	// /status ---------------------------------------------------------------
	"status.title":         "Status",
	"status.uptime":        "Uptime",
	"status.servers":       "Serveurs connectés",
	"status.servers_value": "%d serveurs :\n%s",
	"status.cpu":           "Utilisation CPU",
	"status.memory":        "Utilisation mémoire",
	"status.goroutines":    "Goroutines",
	"status.rss":           "Mémoire RSS",
	"status.log_entries":   "Log entries",
	"status.last_errors":   "5 dernières erreurs",

	// /get-message-history --------------------------------------------------
	"history.title":       "Message History",
	"history.limit_error": "La limite doit être comprise entre 1 et 1000.",
	"history.empty":       "Aucun message en cache pour ce serveur.",
	"history.header":      "Derniers %d messages en cache :\n\n",

	// /get-bot-logs and /get-moderation-logs --------------------------------
	"logs.bot_title": "Bot logs",
	"logs.bot_error": "Erreur lors de la récupération des logs du bot.",
	"logs.bot_empty": "Aucun log trouvé.",
	"logs.mod_title": "Logs de modération",
	"logs.mod_error": "Erreur lors de la récupération des logs de modération.",
	"logs.mod_empty": "Aucun log de modération trouvé.",
	"logs.latest":    "**Derniers logs** (%d/%d)\n\n%s",

	// Selection menus (channels / roles / automod) --------------------------
	"selector.error_channels_title": "Sélection des salons",
	"selector.error_channels":       "Une erreur est survenue lors de la récupération des salons.",
	"selector.error_roles_title":    "Sélection des roles",
	"selector.error_roles":          "Une erreur est survenue lors de la récupération des roles.",
	"selector.no_channel":           "⚠️ Aucun salon sélectionné",
	"selector.no_role":              "⚠️ Aucun role sélectionné.",
	"selector.items_selected":       "**%d**/**%d** item sélectionnés.",
	"selector.updating":             "Mise à jour...",
	"selector.config_done_title":    "Configuration terminée",

	"selector.log.title":       "Sélection des salons de logs",
	"selector.log.description": "Sélectionnez les salons de logs puis cliquez sur \"Terminer\".\nCe sont les salons dans lesquels les actions de modération vont être loggées.",
	"selector.log.placeholder": "Sélectionnez les salons de logs",
	"selector.log.done_header": "✅ Salons de logs sélectionnés :",

	"selector.mod.title":       "Sélection des salons de modération",
	"selector.mod.description": "Sélectionnez les salons de modération puis cliquez sur \"Terminer\".\nCe sont les salons dans lesquels les modérateurs vont être notifiés.",
	"selector.mod.placeholder": "Sélectionnez les salons de modération",
	"selector.mod.done_header": "✅ Salons de modération sélectionnés :",

	"selector.excluded.title":       "Sélection des salons exclus",
	"selector.excluded.description": "Sélectionnez les salons exclus puis cliquez sur \"Terminer\".\nCe sont les salons qui ne seront pas pris en compte dans l'automodération.",
	"selector.excluded.placeholder": "Sélectionnez les salons exclus",
	"selector.excluded.done_header": "✅ Salons exclus sélectionnés :",

	"selector.role.title":       "Sélection des roles de modération",
	"selector.role.description": "Sélectionnez les roles de modération puis cliquez sur \"Terminer\".\nCe sont les roles qui sont administrateurs du serveur et qui seront notifiés.\n\n⚠️ Attention, si vous ne possédez pas au moins un de ces rôles, vous ne pourrez plus utiliser les commandes d'administration !",
	"selector.role.placeholder": "Sélectionnez les roles de modération",
	"selector.role.done":        "✅ %d roles sélectionnés:\n%s",

	"selector.automod.title":       "⚙️ Configuration Automodération",
	"selector.automod.description": "Sélectionnez les fonctionnalités d'automodération à activer puis cliquez sur \"Terminer\".",
	"selector.automod.placeholder": "Sélectionnez les fonctionnalités à activer",
	"selector.automod.none":        "⚠️ Toutes les fonctionnalités d'automodération sont désactivées",
	"selector.automod.done":        "✅ Configuration enregistrée avec succès!\n\n**Fonctionnalités activées** (%d):\n%s",

	"automod_feature.banned_words.name":    "📝 Mots interdits",
	"automod_feature.banned_words.desc":    "Détecte et supprime les messages contenant des mots interdits",
	"automod_feature.banned_websites.name": "🌐 Sites web interdits",
	"automod_feature.banned_websites.desc": "Bloque les liens vers des sites web interdits",
	"automod_feature.spam.name":            "🚨 Détection de spam",
	"automod_feature.spam.desc":            "Bannit automatiquement les utilisateurs qui spamment des messages",

	// Navigation buttons (shared by selectors) ------------------------------
	"button.prev": "◀️ Précédent",
	"button.next": "Suivant ▶️",
	"button.page": "Page %d/%d",
	"button.done": "✅ Terminer",

	// Removal menus (banned word / website) ---------------------------------
	"remove.total":       "**Total** : %d entrée(s). **Page %d/%d**.",
	"remove.notice_none": "⚠️ Aucune entrée n'a été supprimée.",
	"remove.notice":      "✅ %d %s supprimé(s) : %s",
	"remove.type":        "type : %s",

	"remove.word.title":       "🗑️ Supprimer un mot interdit",
	"remove.word.intro":       "Sélectionnez le ou les mots à supprimer de la liste des mots interdits.",
	"remove.word.placeholder": "Sélectionnez les mots à supprimer",
	"remove.word.empty":       "Aucun mot interdit n'est configuré pour ce serveur.",
	"remove.word.error_title": "Mots interdits",
	"remove.word.error":       "Une erreur est survenue lors de la récupération des mots interdits.",
	"remove.word.noun":        "mot(s)",

	"remove.site.title":       "🗑️ Supprimer un site web interdit",
	"remove.site.intro":       "Sélectionnez le ou les sites à supprimer de la liste des sites web interdits.",
	"remove.site.placeholder": "Sélectionnez les sites à supprimer",
	"remove.site.empty":       "Aucun site web interdit n'est configuré pour ce serveur.",
	"remove.site.error_title": "Sites web interdits",
	"remove.site.error":       "Une erreur est survenue lors de la récupération des sites web interdits.",
	"remove.site.noun":        "site(s)",

	// Report action buttons (kick / ban / ignore) ---------------------------
	"report_action.error_title":      "Erreur",
	"report_action.fetch_user_error": "Impossible de récupérer les informations de l'utilisateur.",
	"report_action.kick_dm":          "👢 **Expulsion**\n\nVous avez été expulsé du serveur [%s] le <t:%d:f> pour la raison suivante : \n`Expulsé suite à un signalement`",
	"report_action.ban_dm":           "🔨 **Banissement**\n\nVous avez été banni du serveur [%s] le <t:%d:f> pour la raison suivante : \n`Banni suite à un signalement`",
	"report_action.kicked":           "✅ L'utilisateur **%s** a été expulsé du serveur.",
	"report_action.banned":           "✅ L'utilisateur **%s** a été banni du serveur.",
	"report_action.ignored":          "✅ Le signalement de l'utilisateur **%s** a été ignoré.",
	"report_action.action_error":     "Impossible d'effectuer l'action. Vérifiez que le bot a les permissions nécessaires et que l'utilisateur est toujours sur le serveur.",
	"report_action.done_title":       "Action effectuée",
	"report_action.taken":            "\n\nAction '%s' effectué par %s le <t:%d:f>",
	"report_action.audit_kick":       "Expulsé le %s suite à un signalement",
	"report_action.audit_ban":        "Banni le %s suite à un signalement",

	// /lang -----------------------------------------------------------------
	"lang.menu_title":       "🌐 Langue du serveur",
	"lang.menu_description": "Sélectionnez la langue du bot pour ce serveur.",
	"lang.placeholder":      "Sélectionnez une langue",
	"lang.updated":          "✅ La langue du serveur est maintenant **%s**.",
	"lang.error":            "Une erreur est survenue lors du changement de langue.",

	// Slash-command descriptions (shown in the command picker) --------------
	"cmd.report.desc":                   "Signal un utilisateur à la modération",
	"cmd.report.opt.user":               "L'utilisateur à signaler",
	"cmd.report.opt.reason":             "La raison du signalement",
	"cmd.status.desc":                   "Affiche le statut du bot",
	"cmd.get-message-history.desc":      "Affiche l'historique des messages en cache de la guild",
	"cmd.get-message-history.opt.limit": "Le nombre de messages à récupérer (par défaut 10, maximum 1000)",
	"cmd.get-bot-logs.desc":             "Affiche les derniers logs du bot",
	"cmd.get-moderation-logs.desc":      "Affiche les derniers logs de modération",
	"cmd.set-log-channels.desc":         "Sélectionne les canaux où les rapports de modération seront envoyés",
	"cmd.set-moderation-channels.desc":  "Sélectionne les canaux où les signalements seront envoyés",
	"cmd.set-excluded-channels.desc":    "Sélectionne les canaux exclus de l'automodération",
	"cmd.set-moderation-roles.desc":     "Sélectionne les roles qui auront les permissions de modération",
	"cmd.add-banned-word.desc":          "Ajoute un mot ou une expression à la liste des mots interdits",
	"cmd.add-banned-word.opt.word":      "Le mot ou la regex à interdire",
	"cmd.add-banned-word.opt.is_regex":  "Si le mot est une regex",
	"cmd.get-banned-words.desc":         "Affiche la liste des mots interdits",
	"cmd.remove-banned-word.desc":       "Supprime un ou plusieurs mots de la liste des mots interdits",
	"cmd.add-banned-website.desc":       "Ajoute un site web à la liste des sites interdits",
	"cmd.add-banned-website.opt.url":    "L'URL du site à interdire (ex: example.com)",
	"cmd.get-banned-websites.desc":      "Affiche la liste des sites web interdits",
	"cmd.remove-banned-website.desc":    "Supprime un ou plusieurs sites web de la liste des sites interdits",
	"cmd.configure-automod.desc":        "Active ou désactive les fonctionnalités d'automodération",
	"cmd.help.desc":                     "Liste toutes les commandes disponibles",
	"cmd.lang.desc":                     "Définit la langue du bot pour ce serveur",
}
