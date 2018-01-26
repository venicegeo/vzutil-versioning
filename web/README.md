# Web app section
Rest Endpoints
```
/generate/tags/:org/:repo		    Find dependencies for all tags in a repository
/generate/tags/:org				      Find dependencies for all tags in all known repositories in an organization
/generate/sha/:org/:repo/:sha	  Find dependencies for a repository at specific sha

/report/sha/:org/:repo/:sha		  Create report for a repository at a specific sha
/report/tag/:tag				        Create report for known repositories at specific tag
/report/tag/:org/:tag			      Create report for known repositories in an organization at specific tag
/report/tag/:org/:repo/:tag		  Create report for a repository at specific tag

/list/shas/:org/:repo			      List available shas in a repository
/list/tags/:org/:repo			      List available tags in a repository
/list/tags/:org					        List available tags in known repositories in an organization
/list/projects					        List known projects
/list/projects/:org				      List known projects in an organization
```

For normal users, simply use the `/ui` endpoint in browser
