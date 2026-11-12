package domain

func BuiltinProfiles() []Profile {
	return []Profile{
		{ID: "daily-task", Name: "Daily task", RemindBeforeMinutes: 0, NagMinutes: 15, Builtin: true, Version: 1},
		{ID: "chore", Name: "Chore", RemindBeforeMinutes: 0, NagMinutes: 0, Builtin: true, Version: 1},
		{ID: "birthday", Name: "Birthday", RemindBeforeMinutes: 1440, NagMinutes: 0, Builtin: true, Version: 1},
		{ID: "day-event", Name: "Day event", RemindBeforeMinutes: 60, NagMinutes: 0, Builtin: true, Version: 1},
		{ID: "trip", Name: "Trip", RemindBeforeMinutes: 2880, NagMinutes: 0, Builtin: true, Version: 1},
	}
}
