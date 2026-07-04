package domain

type CommentCreated interface {
	CreatedComment() Comment
	RootComment() (Comment, bool)
	ParentComment() (Comment, bool)
}

type TopLevelCommentCreated struct {
	comment Comment
}

type ReplyCreated struct {
	reply  Comment
	root   Comment
	parent Comment
}

func NewTopLevelCommentCreated(comment Comment) (TopLevelCommentCreated, error) {
	if comment.Status != CommentStatusNormal || !comment.IsTopLevel() {
		return TopLevelCommentCreated{}, ErrCommentNotFound
	}
	return TopLevelCommentCreated{comment: comment}, nil
}

func (e TopLevelCommentCreated) CreatedComment() Comment {
	return e.comment
}

func (e TopLevelCommentCreated) RootComment() (Comment, bool) {
	return Comment{}, false
}

func (e TopLevelCommentCreated) ParentComment() (Comment, bool) {
	return Comment{}, false
}

func NewReplyCreated(reply, root, parent Comment) (ReplyCreated, error) {
	if reply.Status != CommentStatusNormal || !reply.IsReply() {
		return ReplyCreated{}, ErrParentCommentNotFound
	}
	if root.Status != CommentStatusNormal || !root.IsTopLevel() || root.PostID != reply.PostID || reply.RootID != root.ID {
		return ReplyCreated{}, ErrRootCommentNotFound
	}
	if parent.Status != CommentStatusNormal || parent.PostID != reply.PostID || reply.ParentID != parent.ID {
		return ReplyCreated{}, ErrParentCommentNotFound
	}
	if parent.IsTopLevel() && parent.ID != root.ID {
		return ReplyCreated{}, ErrParentCommentNotFound
	}
	if parent.IsReply() && parent.RootID != root.ID {
		return ReplyCreated{}, ErrParentCommentNotFound
	}
	return ReplyCreated{reply: reply, root: root, parent: parent}, nil
}

func (e ReplyCreated) CreatedComment() Comment {
	return e.reply
}

func (e ReplyCreated) RootComment() (Comment, bool) {
	return e.root, true
}

func (e ReplyCreated) ParentComment() (Comment, bool) {
	return e.parent, true
}
