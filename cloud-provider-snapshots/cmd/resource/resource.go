package resource

import (
	"context"
	"fmt"
	"github.com/aws-cloudformation/cloudformation-cli-go-plugin/cfn/encoding"
	"github.com/aws-cloudformation/cloudformation-cli-go-plugin/cfn/handler"
	matlasClient "github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/mongodb/mongodbatlas-cloudformation-resources/util"
)

// Create handles the Create event from the Cloudformation service.
func Create(req handler.Request, prevModel *Model, currentModel *Model) (handler.ProgressEvent, error) {
	client, err := util.CreateMongoDBClient(*currentModel.ApiKeys.PublicKey.Value(), *currentModel.ApiKeys.PrivateKey.Value())
	if err != nil {
		return handler.ProgressEvent{}, err
	}
	if _, ok := req.CallbackContext["status"]; ok {
		currentModel.Id = encoding.NewString(req.CallbackContext["snapshot_id"].(string))
		return validateProgress(client, currentModel, "completed")
	}

	requestParameters := &matlasClient.SnapshotReqPathParameters{
		GroupID:     *currentModel.ProjectId.Value(),
		ClusterName: *currentModel.ClusterName.Value(),
	}
	snapshotRequest := &matlasClient.CloudProviderSnapshot{
		RetentionInDays: int(*currentModel.RetentionInDays.Value()),
		Description:     *currentModel.Description.Value(),
	}

	snapshot, _, err := client.CloudProviderSnapshots.Create(context.Background(), requestParameters, snapshotRequest)
	if err != nil {
		return handler.ProgressEvent{}, fmt.Errorf("error creating cloud provider snapshot: %s", err)
	}

	currentModel.Id = encoding.NewString(snapshot.ID)

	return handler.ProgressEvent{
		OperationStatus:      handler.InProgress,
		Message:              fmt.Sprintf("Create cloud provider snapshots : %s", snapshot.Status),
		ResourceModel:        currentModel,
		CallbackDelaySeconds: 65,
		CallbackContext: map[string]interface{}{
			"status":      snapshot.Status,
			"snapshot_id": snapshot.ID,
		},
	}, nil
}

// Read handles the Read event from the Cloudformation service.
func Read(req handler.Request, prevModel *Model, currentModel *Model) (handler.ProgressEvent, error) {
	client, err := util.CreateMongoDBClient(*currentModel.ApiKeys.PublicKey.Value(), *currentModel.ApiKeys.PrivateKey.Value())
	if err != nil {
		return handler.ProgressEvent{}, err
	}

	projectId := *currentModel.ProjectId.Value()
	snapshotId := *currentModel.Id.Value()
	snapshotRequest := &matlasClient.SnapshotReqPathParameters{
		GroupID:     projectId,
		SnapshotID:  snapshotId,
		ClusterName: *currentModel.ClusterName.Value(),
	}

	snapshot, _, err := client.CloudProviderSnapshots.GetOneCloudProviderSnapshot(context.Background(), snapshotRequest)
	if err != nil {
		return handler.ProgressEvent{}, fmt.Errorf("error reading cloud provider snapshot with id(project: %s, snapshot: %s): %s", projectId, snapshotId, err)
	}

	currentModel.Id = encoding.NewString(snapshot.ID)
	currentModel.Description = encoding.NewString(snapshot.Description)
	currentModel.RetentionInDays = encoding.NewInt(int64(snapshot.RetentionInDays))
	currentModel.Status = encoding.NewString(snapshot.Status)
	currentModel.Type = encoding.NewString(snapshot.Type)
	currentModel.CreatedAt = encoding.NewString(snapshot.CreatedAt)
	currentModel.MasterKeyUuid = encoding.NewString(snapshot.MasterKeyUUID)
	currentModel.MongoVersion = encoding.NewString(snapshot.MongodVersion)
	currentModel.StorageSizeBytes = encoding.NewInt(int64(snapshot.StorageSizeBytes))

	return handler.ProgressEvent{
		OperationStatus: handler.Success,
		Message:         "Read Complete",
		ResourceModel:   currentModel,
	}, nil
}

// Update handles the Update event from the Cloudformation service.
func Update(req handler.Request, prevModel *Model, currentModel *Model) (handler.ProgressEvent, error) {
	// NO-OP
	return handler.ProgressEvent{
		OperationStatus: handler.Success,
		Message:         "Update Complete",
		ResourceModel:   currentModel,
	}, nil
}

// Delete handles the Delete event from the Cloudformation service.
func Delete(req handler.Request, prevModel *Model, currentModel *Model) (handler.ProgressEvent, error) {
	client, err := util.CreateMongoDBClient(*currentModel.ApiKeys.PublicKey.Value(), *currentModel.ApiKeys.PrivateKey.Value())
	if err != nil {
		return handler.ProgressEvent{}, err
	}

	projectId := *currentModel.ProjectId.Value()
	snapshotId := *currentModel.Id.Value()
	snapshotRequest := &matlasClient.SnapshotReqPathParameters{
		GroupID:     projectId,
		SnapshotID:  snapshotId,
		ClusterName: *currentModel.ClusterName.Value(),
	}

	_, err = client.CloudProviderSnapshots.Delete(context.Background(), snapshotRequest)
	if err != nil {
		return handler.ProgressEvent{}, fmt.Errorf("error deleting cloud provider snapshot with id(project: %s, snapshot: %s): %s", projectId, snapshotId, err)
	}

	return handler.ProgressEvent{
		OperationStatus: handler.Success,
		Message:         "Delete Complete",
		ResourceModel:   currentModel,
	}, nil
}

// List handles the List event from the Cloudformation service.
func List(req handler.Request, prevModel *Model, currentModel *Model) (handler.ProgressEvent, error) {
	client, err := util.CreateMongoDBClient(*currentModel.ApiKeys.PublicKey.Value(), *currentModel.ApiKeys.PrivateKey.Value())
	if err != nil {
		return handler.ProgressEvent{}, err
	}

	projectId := *currentModel.ProjectId.Value()
	snapshotRequest := &matlasClient.SnapshotReqPathParameters{
		GroupID:     projectId,
		ClusterName: *currentModel.ClusterName.Value(),
	}

	snapshots, _, err := client.CloudProviderSnapshots.GetAllCloudProviderSnapshots(context.Background(), snapshotRequest)
	if err != nil {
		return handler.ProgressEvent{}, fmt.Errorf("error reading cloud provider snapshot list with id(project: %s): %s", projectId, err)
	}

	var models []Model
	for _, snapshot := range snapshots.Results {
		var model Model
		model.Description = encoding.NewString(snapshot.Description)
		model.RetentionInDays = encoding.NewInt(int64(snapshot.RetentionInDays))
		model.Status = encoding.NewString(snapshot.Status)
		model.Type = encoding.NewString(snapshot.Type)
		model.CreatedAt = encoding.NewString(snapshot.CreatedAt)
		model.MasterKeyUuid = encoding.NewString(snapshot.MasterKeyUUID)
		model.MongoVersion = encoding.NewString(snapshot.MongodVersion)
		model.StorageSizeBytes = encoding.NewInt(int64(snapshot.StorageSizeBytes))
		models = append(models, model)
	}

	return handler.ProgressEvent{
		OperationStatus: handler.Success,
		Message:         "List Complete",
		ResourceModel:   models,
	}, nil
}

func validateProgress(client *matlasClient.Client, currentModel *Model, targetState string) (handler.ProgressEvent, error) {
	isReady, state, err := snapshotIsReady(client, *currentModel.ProjectId.Value(), *currentModel.Id.Value(), *currentModel.ClusterName.Value(), targetState)
	if err != nil {
		return handler.ProgressEvent{}, err
	}

	if !isReady {
		p := handler.NewProgressEvent()
		p.ResourceModel = currentModel
		p.OperationStatus = handler.InProgress
		p.CallbackDelaySeconds = 35
		p.Message = "Pending"
		p.CallbackContext = map[string]interface{}{
			"status":      state,
			"snapshot_id": *currentModel.Id.Value(),
		}
		return p, nil
	}

	p := handler.NewProgressEvent()
	p.ResourceModel = currentModel
	p.OperationStatus = handler.Success
	p.Message = "Complete"
	return p, nil
}

func snapshotIsReady(client *matlasClient.Client, projectId, snapshotId, clusterName, targetState string) (bool, string, error) {
	snapshotRequest := &matlasClient.SnapshotReqPathParameters{
		GroupID:     projectId,
		SnapshotID:  snapshotId,
		ClusterName: clusterName,
	}

	snapshot, resp, err := client.CloudProviderSnapshots.GetOneCloudProviderSnapshot(context.Background(), snapshotRequest)
	if err != nil {
		if snapshot == nil && resp == nil {
			return false, "", err
		}
		if resp != nil && resp.StatusCode == 404 {
			return true, "deleted", nil
		}
		return false, "", err
	}
	return snapshot.Status == targetState, snapshot.Status, nil
}
