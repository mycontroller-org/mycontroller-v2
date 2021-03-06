package importexport

import (
	ml "github.com/mycontroller-org/server/v2/pkg/model"
	backupML "github.com/mycontroller-org/server/v2/pkg/model/backup"
	"github.com/mycontroller-org/server/v2/pkg/utils"
	filterHelper "github.com/mycontroller-org/server/v2/pkg/utils/filter_sort"
	stgType "github.com/mycontroller-org/server/v2/plugin/database/storage/type"
)

// List by filter and pagination
func List(filters []stgType.Filter, pagination *stgType.Pagination) (*stgType.Result, error) {
	files, err := GetBackupFilesList()
	if err != nil {
		return nil, err
	}

	finalList := make([]interface{}, 0)
	totalCount := int64(0)
	if len(files) > 0 {
		if pagination == nil {
			pagination = &stgType.Pagination{
				Limit:  10,
				Offset: 0,
				SortBy: []stgType.Sort{{Field: "id", OrderBy: stgType.SortByASC}},
			}
		}
		sortedFiles, count := filterHelper.Sort(files, pagination)
		totalCount = count
		finalList = filterHelper.Filter(sortedFiles, filters, false)
	}

	result := &stgType.Result{
		Count:  totalCount,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
		Data:   finalList,
	}

	return result, nil
}

// Delete backup files
func Delete(IDs []string) (int64, error) {
	filters := []stgType.Filter{{Key: ml.KeyID, Operator: stgType.OperatorIn, Value: IDs}}

	files, err := GetBackupFilesList()
	if err != nil {
		return 0, err
	}

	finalList := make([]interface{}, 0)
	if len(files) > 0 {
		finalList = filterHelper.Filter(files, filters, false)
	}

	deletedCount := int64(0)
	for _, file := range finalList {
		exportedFile, ok := file.(backupML.BackupFile)
		if !ok {
			continue
		}

		err = utils.RemoveFileOrEmptyDir(exportedFile.FullPath)
		if err != nil {
			return deletedCount, err
		}
		deletedCount++
	}

	return deletedCount, nil
}
