package httpapi

import (
	usercontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/user"
	sharedhttp "github.com/architectcgz/zhicore-go/libs/kit/httpapi"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/application"
	"github.com/gin-gonic/gin"
)

func (h *Handler) batchAvailability(c *gin.Context) {
	if err := requireInternalCaller(c, usercontract.OperationCommentCheckUserAvailability); err != nil {
		writeMappedError(c, err)
		return
	}
	var req usercontract.IDsRequest
	if err := decodeJSONBody(c, &req); err != nil {
		writeValidationError(c)
		return
	}
	items, err := h.service.BatchGetUserAvailability(c.Request.Context(), applicationUserIDs(req.UserIDs))
	if err != nil {
		writeMappedError(c, err)
		return
	}
	resp := usercontract.AvailabilityBatchResponse{Items: make([]usercontract.AvailabilityItem, 0, len(items))}
	for _, item := range items {
		resp.Items = append(resp.Items, usercontract.AvailabilityItem{
			UserID:    int64(item.UserID),
			Available: item.Available,
			Status:    string(item.Status),
		})
	}
	sharedhttp.WriteSuccess(c.Writer, resp)
}

func (h *Handler) batchSimple(c *gin.Context) {
	if err := requireInternalCaller(c, usercontract.OperationCommentBatchGetAuthorSummaries); err != nil {
		writeMappedError(c, err)
		return
	}
	var req usercontract.IDsRequest
	if err := decodeJSONBody(c, &req); err != nil {
		writeValidationError(c)
		return
	}
	result, err := h.service.BatchGetUserSimple(c.Request.Context(), applicationUserIDs(req.UserIDs))
	if err != nil {
		writeMappedError(c, err)
		return
	}
	resp := usercontract.SimpleBatchResponse{
		Items:          make([]usercontract.SimpleUser, 0, len(result.Items)),
		MissingUserIDs: make([]int64, 0, len(result.MissingUserIDs)),
	}
	for _, item := range result.Items {
		resp.Items = append(resp.Items, h.simpleUserResponse(c.Request.Context(), item))
	}
	for _, userID := range result.MissingUserIDs {
		resp.MissingUserIDs = append(resp.MissingUserIDs, int64(userID))
	}
	sharedhttp.WriteSuccess(c.Writer, resp)
}

func (h *Handler) batchCheckBlocked(c *gin.Context) {
	if err := requireInternalCaller(c, usercontract.OperationCommentBatchCheckBlocked); err != nil {
		writeMappedError(c, err)
		return
	}
	var req usercontract.BlockPairsRequest
	if err := decodeJSONBody(c, &req); err != nil {
		writeValidationError(c)
		return
	}
	pairs := make([]application.UserPair, 0, len(req.Pairs))
	for _, pair := range req.Pairs {
		pairs = append(pairs, application.UserPair{
			ActorID:  application.UserID(pair.BlockerID),
			TargetID: application.UserID(pair.BlockedID),
		})
	}
	checked, err := h.service.BatchCheckBlocked(c.Request.Context(), pairs)
	if err != nil {
		writeMappedError(c, err)
		return
	}
	resp := usercontract.BlockPairsResponse{Items: make([]usercontract.BlockPairResult, 0, len(pairs))}
	for _, pair := range pairs {
		resp.Items = append(resp.Items, usercontract.BlockPairResult{
			BlockerID: int64(pair.ActorID),
			BlockedID: int64(pair.TargetID),
			Blocked:   checked[pair],
		})
	}
	sharedhttp.WriteSuccess(c.Writer, resp)
}
