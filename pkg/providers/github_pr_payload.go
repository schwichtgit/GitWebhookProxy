package providers

import "time"

// PullRequestPayload contains the information for GitHub's pull request hook event
type GithubPullRequestPayload struct {
	Action      string `json:"action"`
	Number      int64  `json:"number"`
	PullRequest struct {
		URL      string `json:"url"`
		ID       int64  `json:"id"`
		NodeID   string `json:"node_id"`
		HTMLURL  string `json:"html_url"`
		DiffURL  string `json:"diff_url"`
		PatchURL string `json:"patch_url"`
		IssueURL string `json:"issue_url"`
		Number   int64  `json:"number"`
		State    string `json:"state"`
		Locked   bool   `json:"locked"`
		Title    string `json:"title"`
		User     struct {
			Login             string `json:"login"`
			ID                int64  `json:"id"`
			NodeID            string `json:"node_id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"user"`
		Body           string    `json:"body"`
		CreatedAt      time.Time `json:"created_at"`
		UpdatedAt      time.Time `json:"updated_at"`
		ClosedAt       time.Time `json:"closed_at"`
		MergedAt       time.Time `json:"merged_at"`
		MergeCommitSha string    `json:"merge_commit_sha"`
		RequestedTeams []struct {
			Name            string `json:"body"`
			ID              int64  `json:"id"`
			NodeID          string `json:"node_id"`
			Slug            string `json:"slug"`
			Description     string `json:"description"`
			Privacy         string `json:"privacy"`
			URL             string `json:"url"`
			HTMLURL         string `json:"html_url"`
			MembersURL      string `json:"members_url"`
			RepositoriesURL string `json:"repositories_url"`
			Permission      string `json:"permission"`
		} `json:"requested_teams"`
		Labels []struct {
			ID      int64  `json:"id"`
			NodeID  string `json:"node_id"`
			URL     string `json:"url"`
			Name    string `json:"name"`
			Color   string `json:"color"`
			Default bool   `json:"default"`
		} `json:"labels"`
		CommitsURL        string `json:"commits_url"`
		ReviewCommentsURL string `json:"review_comments_url"`
		ReviewCommentURL  string `json:"review_comment_url"`
		CommentsURL       string `json:"comments_url"`
		StatusesURL       string `json:"statuses_url"`
		Head              struct {
			Label string `json:"label"`
			Ref   string `json:"ref"`
			Sha   string `json:"sha"`
			User  struct {
				Login             string `json:"login"`
				ID                int64  `json:"id"`
				NodeID            string `json:"node_id"`
				AvatarURL         string `json:"avatar_url"`
				GravatarID        string `json:"gravatar_id"`
				URL               string `json:"url"`
				HTMLURL           string `json:"html_url"`
				FollowersURL      string `json:"followers_url"`
				FollowingURL      string `json:"following_url"`
				GistsURL          string `json:"gists_url"`
				StarredURL        string `json:"starred_url"`
				SubscriptionsURL  string `json:"subscriptions_url"`
				OrganizationsURL  string `json:"organizations_url"`
				ReposURL          string `json:"repos_url"`
				EventsURL         string `json:"events_url"`
				ReceivedEventsURL string `json:"received_events_url"`
				Type              string `json:"type"`
				SiteAdmin         bool   `json:"site_admin"`
			} `json:"user"`
			Repo struct {
				ID       int64  `json:"id"`
				NodeID   string `json:"node_id"`
				Name     string `json:"name"`
				FullName string `json:"full_name"`
				Owner    struct {
					Login             string `json:"login"`
					ID                int64  `json:"id"`
					NodeID            string `json:"node_id"`
					AvatarURL         string `json:"avatar_url"`
					GravatarID        string `json:"gravatar_id"`
					URL               string `json:"url"`
					HTMLURL           string `json:"html_url"`
					FollowersURL      string `json:"followers_url"`
					FollowingURL      string `json:"following_url"`
					GistsURL          string `json:"gists_url"`
					StarredURL        string `json:"starred_url"`
					SubscriptionsURL  string `json:"subscriptions_url"`
					OrganizationsURL  string `json:"organizations_url"`
					ReposURL          string `json:"repos_url"`
					EventsURL         string `json:"events_url"`
					ReceivedEventsURL string `json:"received_events_url"`
					Type              string `json:"type"`
					SiteAdmin         bool   `json:"site_admin"`
				} `json:"owner"`
				Private          bool      `json:"private"`
				HTMLURL          string    `json:"html_url"`
				Description      *string   `json:"description"`
				Fork             bool      `json:"fork"`
				URL              string    `json:"url"`
				ForksURL         string    `json:"forks_url"`
				KeysURL          string    `json:"keys_url"`
				CollaboratorsURL string    `json:"collaborators_url"`
				TeamsURL         string    `json:"teams_url"`
				HooksURL         string    `json:"hooks_url"`
				IssueEventsURL   string    `json:"issue_events_url"`
				EventsURL        string    `json:"events_url"`
				AssigneesURL     string    `json:"assignees_url"`
				BranchesURL      string    `json:"branches_url"`
				TagsURL          string    `json:"tags_url"`
				BlobsURL         string    `json:"blobs_url"`
				GitTagsURL       string    `json:"git_tags_url"`
				GitRefsURL       string    `json:"git_refs_url"`
				TreesURL         string    `json:"trees_url"`
				StatusesURL      string    `json:"statuses_url"`
				LanguagesURL     string    `json:"languages_url"`
				StargazersURL    string    `json:"stargazers_url"`
				ContributorsURL  string    `json:"contributors_url"`
				SubscribersURL   string    `json:"subscribers_url"`
				SubscriptionURL  string    `json:"subscription_url"`
				CommitsURL       string    `json:"commits_url"`
				GitCommitsURL    string    `json:"git_commits_url"`
				CommentsURL      string    `json:"comments_url"`
				IssueCommentURL  string    `json:"issue_comment_url"`
				ContentsURL      string    `json:"contents_url"`
				CompareURL       string    `json:"compare_url"`
				MergesURL        string    `json:"merges_url"`
				ArchiveURL       string    `json:"archive_url"`
				DownloadsURL     string    `json:"downloads_url"`
				IssuesURL        string    `json:"issues_url"`
				PullsURL         string    `json:"pulls_url"`
				MilestonesURL    string    `json:"milestones_url"`
				NotificationsURL string    `json:"notifications_url"`
				LabelsURL        string    `json:"labels_url"`
				ReleasesURL      string    `json:"releases_url"`
				DeploymentsURL   string    `json:"deployments_url"`
				CreatedAt        time.Time `json:"created_at"`
				UpdatedAt        time.Time `json:"updated_at"`
				PushedAt         time.Time `json:"pushed_at"`
				GitURL           string    `json:"git_url"`
				SSHURL           string    `json:"ssh_url"`
				CloneURL         string    `json:"clone_url"`
				SvnURL           string    `json:"svn_url"`
				Homepage         *string   `json:"homepage"`
				Size             int64     `json:"size"`
				StargazersCount  int64     `json:"stargazers_count"`
				WatchersCount    int64     `json:"watchers_count"`
				Language         *string   `json:"language"`
				HasIssues        bool      `json:"has_issues"`
				HasProjects      bool      `json:"has_projects"`
				HasDownloads     bool      `json:"has_downloads"`
				HasWiki          bool      `json:"has_wiki"`
				HasPages         bool      `json:"has_pages"`
				ForksCount       int64     `json:"forks_count"`
				MirrorURL        *string   `json:"mirror_url"`
				Archived         bool      `json:"archived"`
				OpenIssuesCount  int64     `json:"open_issues_count"`
				License          struct {
					Key    string `json:"key"`
					Name   string `json:"name"`
					SpdxID string `json:"spdx_id"`
					URL    string `json:"url"`
					NodeID string `json:"node_id"`
				} `json:"license"`
				Forks         int64  `json:"forks"`
				OpenIssues    int64  `json:"open_issues"`
				Watchers      int64  `json:"watchers"`
				DefaultBranch string `json:"default_branch"`
			} `json:"repo"`
		} `json:"head"`
		Base struct {
			Label string `json:"label"`
			Ref   string `json:"ref"`
			Sha   string `json:"sha"`
			User  struct {
				Login             string `json:"login"`
				ID                int64  `json:"id"`
				NodeID            string `json:"node_id"`
				AvatarURL         string `json:"avatar_url"`
				GravatarID        string `json:"gravatar_id"`
				URL               string `json:"url"`
				HTMLURL           string `json:"html_url"`
				FollowersURL      string `json:"followers_url"`
				FollowingURL      string `json:"following_url"`
				GistsURL          string `json:"gists_url"`
				StarredURL        string `json:"starred_url"`
				SubscriptionsURL  string `json:"subscriptions_url"`
				OrganizationsURL  string `json:"organizations_url"`
				ReposURL          string `json:"repos_url"`
				EventsURL         string `json:"events_url"`
				ReceivedEventsURL string `json:"received_events_url"`
				Type              string `json:"type"`
				SiteAdmin         bool   `json:"site_admin"`
			} `json:"user"`
			Repo struct {
				ID       int64  `json:"id"`
				NodeID   string `json:"node_id"`
				Name     string `json:"name"`
				FullName string `json:"full_name"`
				Owner    struct {
					Login             string `json:"login"`
					ID                int64  `json:"id"`
					NodeID            string `json:"node_id"`
					AvatarURL         string `json:"avatar_url"`
					GravatarID        string `json:"gravatar_id"`
					URL               string `json:"url"`
					HTMLURL           string `json:"html_url"`
					FollowersURL      string `json:"followers_url"`
					FollowingURL      string `json:"following_url"`
					GistsURL          string `json:"gists_url"`
					StarredURL        string `json:"starred_url"`
					SubscriptionsURL  string `json:"subscriptions_url"`
					OrganizationsURL  string `json:"organizations_url"`
					ReposURL          string `json:"repos_url"`
					EventsURL         string `json:"events_url"`
					ReceivedEventsURL string `json:"received_events_url"`
					Type              string `json:"type"`
					SiteAdmin         bool   `json:"site_admin"`
				} `json:"owner"`
				Private          bool      `json:"private"`
				HTMLURL          string    `json:"html_url"`
				Description      *string   `json:"description"`
				Fork             bool      `json:"fork"`
				URL              string    `json:"url"`
				ForksURL         string    `json:"forks_url"`
				KeysURL          string    `json:"keys_url"`
				CollaboratorsURL string    `json:"collaborators_url"`
				TeamsURL         string    `json:"teams_url"`
				HooksURL         string    `json:"hooks_url"`
				IssueEventsURL   string    `json:"issue_events_url"`
				EventsURL        string    `json:"events_url"`
				AssigneesURL     string    `json:"assignees_url"`
				BranchesURL      string    `json:"branches_url"`
				TagsURL          string    `json:"tags_url"`
				BlobsURL         string    `json:"blobs_url"`
				GitTagsURL       string    `json:"git_tags_url"`
				GitRefsURL       string    `json:"git_refs_url"`
				TreesURL         string    `json:"trees_url"`
				StatusesURL      string    `json:"statuses_url"`
				LanguagesURL     string    `json:"languages_url"`
				StargazersURL    string    `json:"stargazers_url"`
				ContributorsURL  string    `json:"contributors_url"`
				SubscribersURL   string    `json:"subscribers_url"`
				SubscriptionURL  string    `json:"subscription_url"`
				CommitsURL       string    `json:"commits_url"`
				GitCommitsURL    string    `json:"git_commits_url"`
				CommentsURL      string    `json:"comments_url"`
				IssueCommentURL  string    `json:"issue_comment_url"`
				ContentsURL      string    `json:"contents_url"`
				CompareURL       string    `json:"compare_url"`
				MergesURL        string    `json:"merges_url"`
				ArchiveURL       string    `json:"archive_url"`
				DownloadsURL     string    `json:"downloads_url"`
				IssuesURL        string    `json:"issues_url"`
				PullsURL         string    `json:"pulls_url"`
				MilestonesURL    string    `json:"milestones_url"`
				NotificationsURL string    `json:"notifications_url"`
				LabelsURL        string    `json:"labels_url"`
				ReleasesURL      string    `json:"releases_url"`
				DeploymentsURL   string    `json:"deployments_url"`
				CreatedAt        time.Time `json:"created_at"`
				UpdatedAt        time.Time `json:"updated_at"`
				PushedAt         time.Time `json:"pushed_at"`
				GitURL           string    `json:"git_url"`
				SSHURL           string    `json:"ssh_url"`
				CloneURL         string    `json:"clone_url"`
				SvnURL           string    `json:"svn_url"`
				Homepage         *string   `json:"homepage"`
				Size             int64     `json:"size"`
				StargazersCount  int64     `json:"stargazers_count"`
				WatchersCount    int64     `json:"watchers_count"`
				Language         *string   `json:"language"`
				HasIssues        bool      `json:"has_issues"`
				HasProjects      bool      `json:"has_projects"`
				HasDownloads     bool      `json:"has_downloads"`
				HasWiki          bool      `json:"has_wiki"`
				HasPages         bool      `json:"has_pages"`
				ForksCount       int64     `json:"forks_count"`
				MirrorURL        *string   `json:"mirror_url"`
				Archived         bool      `json:"archived"`
				OpenIssuesCount  int64     `json:"open_issues_count"`
				License          struct {
					Key    string `json:"key"`
					Name   string `json:"name"`
					SpdxID string `json:"spdx_id"`
					URL    string `json:"url"`
					NodeID string `json:"node_id"`
				} `json:"license"`
				Forks         int64  `json:"forks"`
				OpenIssues    int64  `json:"open_issues"`
				Watchers      int64  `json:"watchers"`
				DefaultBranch string `json:"default_branch"`
			} `json:"repo"`
		} `json:"base"`
		Links struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
			Issue struct {
				Href string `json:"href"`
			} `json:"issue"`
			Comments struct {
				Href string `json:"href"`
			} `json:"comments"`
			ReviewComments struct {
				Href string `json:"href"`
			} `json:"review_comments"`
			ReviewComment struct {
				Href string `json:"href"`
			} `json:"review_comment"`
			Commits struct {
				Href string `json:"href"`
			} `json:"commits"`
			Statuses struct {
				Href string `json:"href"`
			} `json:"statuses"`
		} `json:"_links"`
		AuthorAssociation string `json:"author_association"`
		Merged            bool   `json:"merged"`
		Mergeable         bool   `json:"mergeable"`
		Rebaseable        bool   `json:"rebaseable"`
		MergeableState    string `json:"mergeable_state"`
		MergedBy          struct {
			Login             string `json:"login"`
			ID                int64  `json:"id"`
			NodeID            string `json:"node_id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"merged_by"`
		Comments            int64 `json:"comments"`
		ReviewComments      int64 `json:"review_comments"`
		MaintainerCanModify bool  `json:"maintainer_can_modify"`
		Commits             int64 `json:"commits"`
		Additions           int64 `json:"additions"`
		Deletions           int64 `json:"deletions"`
		ChangedFiles        int64 `json:"changed_files"`
	} `json:"pull_request"`
	Repository struct {
		ID       int64  `json:"id"`
		NodeID   string `json:"node_id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Owner    struct {
			Login             string `json:"login"`
			ID                int64  `json:"id"`
			NodeID            string `json:"node_id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"owner"`
		Private          bool      `json:"private"`
		HTMLURL          string    `json:"html_url"`
		Description      *string   `json:"description"`
		Fork             bool      `json:"fork"`
		URL              string    `json:"url"`
		ForksURL         string    `json:"forks_url"`
		KeysURL          string    `json:"keys_url"`
		CollaboratorsURL string    `json:"collaborators_url"`
		TeamsURL         string    `json:"teams_url"`
		HooksURL         string    `json:"hooks_url"`
		IssueEventsURL   string    `json:"issue_events_url"`
		EventsURL        string    `json:"events_url"`
		AssigneesURL     string    `json:"assignees_url"`
		BranchesURL      string    `json:"branches_url"`
		TagsURL          string    `json:"tags_url"`
		BlobsURL         string    `json:"blobs_url"`
		GitTagsURL       string    `json:"git_tags_url"`
		GitRefsURL       string    `json:"git_refs_url"`
		TreesURL         string    `json:"trees_url"`
		StatusesURL      string    `json:"statuses_url"`
		LanguagesURL     string    `json:"languages_url"`
		StargazersURL    string    `json:"stargazers_url"`
		ContributorsURL  string    `json:"contributors_url"`
		SubscribersURL   string    `json:"subscribers_url"`
		SubscriptionURL  string    `json:"subscription_url"`
		CommitsURL       string    `json:"commits_url"`
		GitCommitsURL    string    `json:"git_commits_url"`
		CommentsURL      string    `json:"comments_url"`
		IssueCommentURL  string    `json:"issue_comment_url"`
		ContentsURL      string    `json:"contents_url"`
		CompareURL       string    `json:"compare_url"`
		MergesURL        string    `json:"merges_url"`
		ArchiveURL       string    `json:"archive_url"`
		DownloadsURL     string    `json:"downloads_url"`
		IssuesURL        string    `json:"issues_url"`
		PullsURL         string    `json:"pulls_url"`
		MilestonesURL    string    `json:"milestones_url"`
		NotificationsURL string    `json:"notifications_url"`
		LabelsURL        string    `json:"labels_url"`
		ReleasesURL      string    `json:"releases_url"`
		DeploymentsURL   string    `json:"deployments_url"`
		CreatedAt        time.Time `json:"created_at"`
		UpdatedAt        time.Time `json:"updated_at"`
		PushedAt         time.Time `json:"pushed_at"`
		GitURL           string    `json:"git_url"`
		SSHURL           string    `json:"ssh_url"`
		CloneURL         string    `json:"clone_url"`
		SvnURL           string    `json:"svn_url"`
		Homepage         *string   `json:"homepage"`
		Size             int64     `json:"size"`
		StargazersCount  int64     `json:"stargazers_count"`
		WatchersCount    int64     `json:"watchers_count"`
		Language         *string   `json:"language"`
		HasIssues        bool      `json:"has_issues"`
		HasProjects      bool      `json:"has_projects"`
		HasDownloads     bool      `json:"has_downloads"`
		HasWiki          bool      `json:"has_wiki"`
		HasPages         bool      `json:"has_pages"`
		ForksCount       int64     `json:"forks_count"`
		MirrorURL        *string   `json:"mirror_url"`
		Archived         bool      `json:"archived"`
		OpenIssuesCount  int64     `json:"open_issues_count"`
		License          struct {
			Key    string `json:"key"`
			Name   string `json:"name"`
			SpdxID string `json:"spdx_id"`
			URL    string `json:"url"`
			NodeID string `json:"node_id"`
		} `json:"license"`
		Forks         int64  `json:"forks"`
		OpenIssues    int64  `json:"open_issues"`
		Watchers      int64  `json:"watchers"`
		DefaultBranch string `json:"default_branch"`
	} `json:"repository"`
	Organization struct {
		Login            string `json:"login"`
		ID               int64  `json:"id"`
		NodeID           string `json:"node_id"`
		URL              string `json:"url"`
		ReposURL         string `json:"repos_url"`
		EventsURL        string `json:"events_url"`
		HooksURL         string `json:"hooks_url"`
		IssuesURL        string `json:"issues_url"`
		MembersURL       string `json:"members_url"`
		PublicMembersURL string `json:"public_members_url"`
		AvatarURL        string `json:"avatar_url"`
		Description      string `json:"description"`
	} `json:"organization"`
	Sender struct {
		Login             string `json:"login"`
		ID                int64  `json:"id"`
		NodeID            string `json:"node_id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"sender"`
}
