package controlplane

import "ad-event-processor/internal/commandpalette"

func (s *Service) CommandPaletteService() *commandpalette.Service {
	if s == nil || s.pool == nil {
		return commandpalette.NewService(nil)
	}
	return commandpalette.NewService(s.pool)
}
