package task

import (
	"sync"

	taskML "github.com/mycontroller-org/backend/v2/pkg/model/task"
	helper "github.com/mycontroller-org/backend/v2/pkg/utils/filter_sort"
	rsUtils "github.com/mycontroller-org/backend/v2/pkg/utils/resource_service"
	stgml "github.com/mycontroller-org/backend/v2/plugin/storage"
	"go.uber.org/zap"
)

type store struct {
	tasks map[string]taskML.Config
	mutex sync.Mutex
}

var tasksStore = store{
	tasks: make(map[string]taskML.Config),
}

// Add a task
func (s *store) Add(task taskML.Config) {
	if !task.Enabled {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.tasks[task.ID] = task

}

// UpdateState of a task
func (s *store) UpdateState(id string, state *taskML.State) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if task, ok := s.tasks[id]; ok {
		task.State = state
	}
	rsUtils.SetTaskState(id, *state)
}

// Remove a task
func (s *store) Remove(id string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.tasks, id)
}

// GetByID returns handler by id
func (s *store) Get(id string) taskML.Config {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.tasks[id]
}

func (s *store) RemoveAll() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tasksStore.tasks = make(map[string]taskML.Config)
}

func (s *store) ListIDs() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	ids := make([]string, 0)
	for id := range s.tasks {
		ids = append(ids, id)
	}
	return ids
}

func (s *store) filterTasks(resource *resourceWrapper) []taskML.Config {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	filteredTasks := make([]taskML.Config, 0)
	for id := range s.tasks {
		task := s.tasks[id]
		filters := s.getFilters(task.EventFilter.Selectors)
		matching := false
		zap.L().Debug("filterTasks", zap.Any("filters", filters), zap.Any("resource", resource.Resource))

		if len(filters) == 0 {
			matching = true
		} else {
			zap.L().Debug("filterTasks", zap.Any("filters", filters), zap.Any("resource", resource.Resource))
			matching = helper.IsMatching(resource.Resource, filters)
		}
		if matching {
			filteredTasks = append(filteredTasks, task)
		}
	}
	return filteredTasks
}

func (s *store) getFilters(filtersMap map[string]string) []stgml.Filter {
	filters := make([]stgml.Filter, 0)
	for k, v := range filtersMap {
		filters = append(filters, stgml.Filter{Key: k, Operator: stgml.OperatorEqual, Value: v})
	}
	return filters
}