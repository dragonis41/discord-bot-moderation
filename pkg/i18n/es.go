package i18n

// es is the Spanish catalog.
var es = map[string]string{
	// Footers ---------------------------------------------------------------
	"footer.hint":            "💡 Consejo: Usa /help para ver los comandos disponibles.",
	"footer.selection_saved": "Las selecciones se guardan con cada cambio.",

	// /report ---------------------------------------------------------------
	"report.title":           "Reporte",
	"report.user_not_found":  "No se encontró al usuario especificado. Verifica el ID e inténtalo de nuevo.",
	"report.no_mod_roles":    "No hay ningún rol de moderación configurado para este servidor. Contacta a un administrador.",
	"report.error_generic":   "Ocurrió un error al procesar tu solicitud. Contacta a un moderador.",
	"report.no_mod_channels": "No hay ningún canal de moderación configurado para este servidor. Contacta a un administrador.",
	"report.description":     "**Usuario reportado**: <@!%s> (%s | %s)\n**Canal**: <#%s>\n**Motivo**: %s\n\n**Mensajes recientes**:\n%s",
	"report.alert_title":     "🚨 Nuevo reporte de %s",
	"report.alert_footer":    "Aún no se ha tomado ninguna acción.",
	"report.received":        "El usuario %s ha sido reportado al equipo de moderación.",
	"report.received_footer": "Gracias por ayudar a mantener este servidor sano.",
	"report.no_reason":       "Sin motivo especificado",

	// /help -----------------------------------------------------------------
	"help.title":          "💡 Ayuda",
	"help.description":    "Esta es la lista de comandos disponibles:",
	"help.options_label":  "**Opciones**:",
	"help.required_label": "**Obligatorio:**",
	"help.optional_label": "**Opcional:**",

	// Automod (DM warning + escalation) -------------------------------------
	"automod.warning":               "⚠️ **Advertencia %d/%d**\nTu mensaje en el servidor [%s] fue eliminado porque %s\n\nAquí tienes una copia de tu mensaje:\n",
	"automod.cause_banned_word":     "contiene la palabra `%s`",
	"automod.cause_banned_website":  "contiene un enlace prohibido: `%s`",
	"automod.reason_banned_word":    "La palabra `%s` está prohibida",
	"automod.reason_banned_website": "El sitio `%s` está prohibido",
	"automod.violation_suffix":      "%s (infracción %d/%d)",
	"automod.ban_reason":            "%s\n%d infracciones de automoderación",
	"automod.spam_reason":           "Spam (mensaje repetido %d veces en %d canales en %s)",

	// Automod alert (sent to log channels) ----------------------------------
	"automod.alert.title":       "🚨 Acción de automoderación activada",
	"automod.alert.description": "**Usuario**: <@%s> (%s | %s)\n**Acción**: `%s`\n**Regla activada**: `%s`\n**Motivo**: %s\n\n**Mensajes recientes**: \n%s",

	// Moderation actions (kick/ban DMs) -------------------------------------
	"action.kick_dm": "👢 **Expulsión**\n\nHas sido expulsado del servidor [%s] por el siguiente motivo: \n*%s*",
	"action.ban_dm":  "🔨 **Baneo**\n\nHas sido baneado del servidor [%s] por el siguiente motivo: \n*%s*",

	// Recent-message rendering ----------------------------------------------
	"msg.files_embeds":   "<archivo(s): %d + embed(s)>",
	"msg.file":           "<archivo>",
	"msg.files":          "<%d archivos>",
	"msg.embed":          "<embed>",
	"msg.sticker":        "<sticker>",
	"msg.stickers":       "<%d stickers>",
	"msg.empty":          "<mensaje vacío>",
	"msg.attachment_one": " [+archivo]",
	"msg.attachments":    " [+%d archivos]",
	"msg.more":           "... y %d mensaje(s) más",
	"msg.none_recent":    "*No hay mensajes recientes disponibles*",

	// Banned words ----------------------------------------------------------
	"bannedword.title":            "Palabra prohibida",
	"bannedword.empty_word":       "La palabra no puede estar vacía.",
	"bannedword.add_error":        "Ocurrió un error al añadir la palabra prohibida.",
	"bannedword.type_literal":     "literal",
	"bannedword.type_regex":       "regex",
	"bannedword.added_title":      "✅ Palabra prohibida añadida",
	"bannedword.added":            "La palabra `%s` (tipo: %s) ha sido añadida a la lista de palabras prohibidas.",
	"bannedword.list_error_title": "Palabras prohibidas",
	"bannedword.list_error":       "Ocurrió un error al obtener las palabras prohibidas.",
	"bannedword.list_empty_title": "📝 Palabras prohibidas",
	"bannedword.list_empty":       "No hay ninguna palabra prohibida configurada para este servidor.",
	"bannedword.list_item":        "• **ID %d**: `%s` (tipo: %s)\n",
	"bannedword.list_title":       "📝 Lista de palabras prohibidas",
	"bannedword.list_total":       "**Total**: %d palabra(s) prohibida(s)\n\n%s",

	// Banned websites -------------------------------------------------------
	"bannedsite.title":            "Sitio web prohibido",
	"bannedsite.empty_url":        "La URL no puede estar vacía.",
	"bannedsite.add_error":        "Ocurrió un error al añadir el sitio web prohibido.",
	"bannedsite.added_title":      "✅ Sitio web prohibido añadido",
	"bannedsite.added":            "El sitio `%s` ha sido añadido a la lista de sitios web prohibidos.",
	"bannedsite.list_error_title": "Sitios web prohibidos",
	"bannedsite.list_error":       "Ocurrió un error al obtener los sitios web prohibidos.",
	"bannedsite.list_empty_title": "🌐 Sitios web prohibidos",
	"bannedsite.list_empty":       "No hay ningún sitio web prohibido configurado para este servidor.",
	"bannedsite.list_item":        "• **ID %d**: `%s`\n",
	"bannedsite.list_title":       "🌐 Lista de sitios web prohibidos",
	"bannedsite.list_total":       "**Total**: %d sitio(s) web prohibido(s)\n\n%s",

	// Configure automod -----------------------------------------------------
	"automodcfg.error_title": "Configuración de automoderación",
	"automodcfg.error":       "Ocurrió un error al obtener las funciones.",

	// /status ---------------------------------------------------------------
	"status.title":         "Estado",
	"status.uptime":        "Tiempo activo",
	"status.servers":       "Servidores conectados",
	"status.servers_value": "%d servidores:\n%s",
	"status.cpu":           "Uso de CPU",
	"status.memory":        "Uso de memoria",
	"status.goroutines":    "Goroutines",
	"status.rss":           "Memoria RSS",
	"status.log_entries":   "Entradas de registro",
	"status.last_errors":   "Últimos 5 errores",

	// /get-message-history --------------------------------------------------
	"history.title":       "Historial de mensajes",
	"history.limit_error": "El límite debe estar entre 1 y 100.",
	"history.empty":       "No hay mensajes en caché para este servidor.",
	"history.header":      "Últimos %d mensajes en caché:\n\n",

	// /get-bot-logs and /get-moderation-logs --------------------------------
	"logs.bot_title": "Registros del bot",
	"logs.bot_error": "Error al obtener los registros del bot.",
	"logs.bot_empty": "No se encontró ningún registro.",
	"logs.mod_title": "Registros de moderación",
	"logs.mod_error": "Error al obtener los registros de moderación.",
	"logs.mod_empty": "No se encontró ningún registro de moderación.",
	"logs.latest":    "**Últimos registros** (%d/%d)\n\n%s",

	// Selection menus (channels / roles / automod) --------------------------
	"selector.error_channels_title": "Selección de canales",
	"selector.error_channels":       "Ocurrió un error al obtener los canales.",
	"selector.error_roles_title":    "Selección de roles",
	"selector.error_roles":          "Ocurrió un error al obtener los roles.",
	"selector.no_channel":           "⚠️ Ningún canal seleccionado",
	"selector.no_role":              "⚠️ Ningún rol seleccionado.",
	"selector.items_selected":       "**%d**/**%d** elemento(s) seleccionado(s).",
	"selector.updating":             "Actualizando...",
	"selector.config_done_title":    "Configuración completada",

	"selector.log.title":       "Selección de canales de registro",
	"selector.log.description": "Selecciona los canales de registro y luego haz clic en \"Hecho\".\nSon los canales donde se registrarán las acciones de moderación.",
	"selector.log.placeholder": "Selecciona los canales de registro",
	"selector.log.done_header": "✅ Canales de registro seleccionados:",

	"selector.mod.title":       "Selección de canales de moderación",
	"selector.mod.description": "Selecciona los canales de moderación y luego haz clic en \"Hecho\".\nSon los canales donde se notificará a los moderadores.",
	"selector.mod.placeholder": "Selecciona los canales de moderación",
	"selector.mod.done_header": "✅ Canales de moderación seleccionados:",

	"selector.excluded.title":       "Selección de canales excluidos",
	"selector.excluded.description": "Selecciona los canales excluidos y luego haz clic en \"Hecho\".\nSon los canales que la automoderación ignorará.",
	"selector.excluded.placeholder": "Selecciona los canales excluidos",
	"selector.excluded.done_header": "✅ Canales excluidos seleccionados:",

	"selector.role.title":       "Selección de roles de moderación",
	"selector.role.description": "Selecciona los roles de moderación y luego haz clic en \"Hecho\".\nSon los roles que administran el servidor y serán notificados.\n\n⚠️ Atención: si no tienes al menos uno de estos roles, ¡ya no podrás usar los comandos de administración!",
	"selector.role.placeholder": "Selecciona los roles de moderación",
	"selector.role.done":        "✅ %d roles seleccionados:\n%s",

	"selector.automod.title":       "⚙️ Configuración de automoderación",
	"selector.automod.description": "Selecciona las funciones de automoderación a activar y luego haz clic en \"Hecho\".",
	"selector.automod.placeholder": "Selecciona las funciones a activar",
	"selector.automod.none":        "⚠️ Todas las funciones de automoderación están desactivadas",
	"selector.automod.done":        "✅ ¡Configuración guardada con éxito!\n\n**Funciones activadas** (%d):\n%s",

	"automod_feature.banned_words.name":    "📝 Palabras prohibidas",
	"automod_feature.banned_words.desc":    "Detecta y elimina los mensajes que contienen palabras prohibidas",
	"automod_feature.banned_websites.name": "🌐 Sitios web prohibidos",
	"automod_feature.banned_websites.desc": "Bloquea los enlaces a sitios web prohibidos",
	"automod_feature.spam.name":            "🚨 Detección de spam",
	"automod_feature.spam.desc":            "Banea automáticamente a los usuarios que envían spam",

	// Navigation buttons (shared by selectors) ------------------------------
	"button.prev": "◀️ Anterior",
	"button.next": "Siguiente ▶️",
	"button.page": "Página %d/%d",
	"button.done": "✅ Hecho",

	// Removal menus (banned word / website) ---------------------------------
	"remove.total":       "**Total**: %d entrada(s). **Página %d/%d**.",
	"remove.notice_none": "⚠️ No se eliminó ninguna entrada.",
	"remove.notice":      "✅ %d %s eliminada(s): %s",
	"remove.type":        "tipo: %s",

	"remove.word.title":       "🗑️ Eliminar una palabra prohibida",
	"remove.word.intro":       "Selecciona la(s) palabra(s) a eliminar de la lista de palabras prohibidas.",
	"remove.word.placeholder": "Selecciona las palabras a eliminar",
	"remove.word.empty":       "No hay ninguna palabra prohibida configurada para este servidor.",
	"remove.word.error_title": "Palabras prohibidas",
	"remove.word.error":       "Ocurrió un error al obtener las palabras prohibidas.",
	"remove.word.noun":        "palabra(s)",

	"remove.site.title":       "🗑️ Eliminar un sitio web prohibido",
	"remove.site.intro":       "Selecciona el/los sitio(s) a eliminar de la lista de sitios web prohibidos.",
	"remove.site.placeholder": "Selecciona los sitios a eliminar",
	"remove.site.empty":       "No hay ningún sitio web prohibido configurado para este servidor.",
	"remove.site.error_title": "Sitios web prohibidos",
	"remove.site.error":       "Ocurrió un error al obtener los sitios web prohibidos.",
	"remove.site.noun":        "sitio(s)",

	// Report action buttons (kick / ban / ignore) ---------------------------
	"report_action.error_title":      "Error",
	"report_action.fetch_user_error": "No se pudo obtener la información del usuario.",
	"report_action.kick_dm":          "👢 **Expulsión**\n\nHas sido expulsado del servidor [%s] el <t:%d:f> por el siguiente motivo: \n`Expulsado tras un reporte`",
	"report_action.ban_dm":           "🔨 **Baneo**\n\nHas sido baneado del servidor [%s] el <t:%d:f> por el siguiente motivo: \n`Baneado tras un reporte`",
	"report_action.kicked":           "✅ El usuario **%s** ha sido expulsado del servidor.",
	"report_action.banned":           "✅ El usuario **%s** ha sido baneado del servidor.",
	"report_action.ignored":          "✅ El reporte del usuario **%s** ha sido ignorado.",
	"report_action.action_error":     "No se pudo realizar la acción. Verifica que el bot tenga los permisos necesarios y que el usuario siga en el servidor.",
	"report_action.done_title":       "Acción realizada",
	"report_action.taken":            "\n\nAcción '%s' realizada por %s el <t:%d:f>",
	"report_action.audit_kick":       "Expulsado el %s tras un reporte",
	"report_action.audit_ban":        "Baneado el %s tras un reporte",

	// /lang -----------------------------------------------------------------
	"lang.menu_title":       "🌐 Idioma del servidor",
	"lang.menu_description": "Selecciona el idioma del bot para este servidor.",
	"lang.placeholder":      "Selecciona un idioma",
	"lang.updated":          "✅ El idioma del servidor ahora es **%s**.",
	"lang.error":            "Ocurrió un error al cambiar el idioma.",

	// Slash-command descriptions (shown in the command picker) --------------
	"cmd.report.desc":                   "Reporta a un usuario al equipo de moderación",
	"cmd.report.opt.user":               "El usuario a reportar",
	"cmd.report.opt.reason":             "El motivo del reporte",
	"cmd.status.desc":                   "Muestra el estado del bot",
	"cmd.get-message-history.desc":      "Muestra el historial de mensajes en caché del servidor",
	"cmd.get-message-history.opt.limit": "Número de mensajes a obtener (por defecto 10, máximo 100)",
	"cmd.get-bot-logs.desc":             "Muestra los últimos registros del bot",
	"cmd.get-moderation-logs.desc":      "Muestra los últimos registros de moderación",
	"cmd.set-log-channels.desc":         "Selecciona los canales donde se enviarán los reportes de moderación",
	"cmd.set-moderation-channels.desc":  "Selecciona los canales donde se enviarán los reportes",
	"cmd.set-excluded-channels.desc":    "Selecciona los canales excluidos de la automoderación",
	"cmd.set-moderation-roles.desc":     "Selecciona los roles que tendrán permisos de moderación",
	"cmd.add-banned-word.desc":          "Añade una palabra o expresión a la lista de palabras prohibidas",
	"cmd.add-banned-word.opt.word":      "La palabra o regex a prohibir",
	"cmd.add-banned-word.opt.is_regex":  "Si la palabra es una regex",
	"cmd.get-banned-words.desc":         "Muestra la lista de palabras prohibidas",
	"cmd.remove-banned-word.desc":       "Elimina una o varias palabras de la lista de palabras prohibidas",
	"cmd.add-banned-website.desc":       "Añade un sitio web a la lista de sitios prohibidos",
	"cmd.add-banned-website.opt.url":    "La URL del sitio a prohibir (ej.: example.com)",
	"cmd.get-banned-websites.desc":      "Muestra la lista de sitios web prohibidos",
	"cmd.remove-banned-website.desc":    "Elimina uno o varios sitios web de la lista de sitios prohibidos",
	"cmd.configure-automod.desc":        "Activa o desactiva las funciones de automoderación",
	"cmd.help.desc":                     "Lista todos los comandos disponibles",
	"cmd.lang.desc":                     "Establece el idioma del bot para este servidor",
}
