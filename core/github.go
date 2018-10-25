package core

import (
  "context"
  "strings"
  "time"
  "fmt"
  "github.com/google/go-github/github"
)

type GithubOwner struct {
  Login     *string
  ID        *int64
  Type      *string
  Name      *string
  AvatarURL *string
  URL       *string
  Company   *string
  Blog      *string
  Location  *string
  Email     *string
  Bio       *string
}

type GithubRepository struct {
  Owner         *string
  ID            *int64
  Name          *string
  FullName      *string
  CloneURL      *string
  URL           *string
  DefaultBranch *string
  Description   *string
  Homepage      *string
}

func GetUserOrOrganization(login string, client *github.Client) (*GithubOwner, error) {
  ctx := context.Background()
  user, _, err := client.Users.Get(ctx, login)
  if err != nil {
    return nil, err
  }
  return &GithubOwner{
    Login:     user.Login,
    ID:        user.ID,
    Type:      user.Type,
    Name:      user.Name,
    AvatarURL: user.AvatarURL,
    URL:       user.HTMLURL,
    Company:   user.Company,
    Blog:      user.Blog,
    Location:  user.Location,
    Email:     user.Email,
    Bio:       user.Bio,
  }, nil
}

func GetRepositoriesFromOwner(login *string, client *github.Client) ([]*GithubRepository, error) {
  var allRepos []*GithubRepository
  loginVal := *login
  ctx := context.Background()
  opt := &github.RepositoryListOptions{
    Type: "sources",
  }

  for {
    repos, resp, err := client.Repositories.List(ctx, loginVal, opt)
    if err != nil {
      return allRepos, err
    }
    for _, repo := range repos {
      if !*repo.Fork {
        r := GithubRepository{
          Owner:         repo.Owner.Login,
          ID:            repo.ID,
          Name:          repo.Name,
          FullName:      repo.FullName,
          CloneURL:      repo.CloneURL,
          URL:           repo.HTMLURL,
          DefaultBranch: repo.DefaultBranch,
          Description:   repo.Description,
          Homepage:      repo.Homepage,
        }
        allRepos = append(allRepos, &r)
      }
    }
    if resp.NextPage == 0 {
      break
    }
    opt.Page = resp.NextPage
  }

  return allRepos, nil
}

func GetOrganizationMembers(login *string, client *github.Client) ([]*GithubOwner, error) {
  var allMembers []*GithubOwner
  loginVal := *login
  ctx := context.Background()
  opt := &github.ListMembersOptions{}
  for {
    members, resp, err := client.Organizations.ListMembers(ctx, loginVal, opt)
    if err != nil {
      return allMembers, err
    }
    for _, member := range members {
      allMembers = append(allMembers, &GithubOwner{Login: member.Login, ID: member.ID, Type: member.Type})
    }
    if resp.NextPage == 0 {
      break
    }
    opt.Page = resp.NextPage
  }
  return allMembers, nil
}

func GetUserCount(client *github.Client) (int64, error){
  return 44452275, nil
  ctx := context.Background()
  opt := &github.UserListOptions{
    Since: 0,
  }

  sinceVal := int64(0)
  lastVal := int64(0)

  for {
    users, _, err := client.Users.ListAll(ctx, opt)
    if err != nil {
      return -1, err
    }
    for _, user := range users {
      sinceVal = int64(*user.ID)
    }
    if len(users) == 0 {
      if sinceVal == lastVal { 
        return int64(sinceVal), nil
      }
      sinceVal = (lastVal + sinceVal) / 2 
    } else {
      lastVal = sinceVal
      sinceVal *= 2
    }
    opt.Since = int64(sinceVal)
  }

  return int64(sinceVal), nil
}

func ParseTime(GHTime string) (time.Time, error){
  //Not parsing correctly. Need to convert GH time to UTC
  //"2018-10-25 17:56:28 -0500 CDT"
  //Goal is RFC3339 -> "2006-01-02T15:04:05Z07:00"
  sections := strings.Fields(GHTime)
  date := sections[0]
  clock := sections[1]
  timezone := sections[2]

  zone := timezone[0:3] + ":" + timezone[3:5]



  rfcTime := fmt.Sprintf("%sT%s%s", date, clock, zone)
  fmt.Printf("RFC Time is %s\n", rfcTime)

  t, err := time.Parse(time.RFC3339, rfcTime)
  if err != nil {
    fmt.Printf("err: %s\n", err)
  }

  fmt.Printf("Hour: %d\nMinute: %d\n", t.Hour(), t.Minute())

  return t, nil
}

func GetAllUsers(start int64, end int64, client *github.Client) ([]*GithubOwner, error) {
  var allUsers []*GithubOwner
  ctx := context.Background()
  opt := &github.UserListOptions{
    Since: start,
  }


  rateLimit, _, _ := client.RateLimits(ctx)
  fmt.Printf("Core: %s\n", rateLimit.Core.String())

  sinceVal := int64(start)

  for {
    users, _, err := client.Users.ListAll(ctx, opt)
    if err != nil {
      if _, ok := err.(*github.RateLimitError); ok {
        return nil, err
      } else {
        //We hit the rate limit. Time to go sleep!
        time.Sleep(10 * time.Minute)
        users, _, err = client.Users.ListAll(ctx, opt)
      }
    }
    for _, user := range users {
      owner := GithubOwner{
        Login:     user.Login,
        ID:        user.ID,
        Type:      user.Type,
        Name:      user.Name,
        AvatarURL: user.AvatarURL,
        URL:       user.HTMLURL,
        Company:   user.Company,
        Blog:      user.Blog,
        Location:  user.Location,
        Email:     user.Email,
        Bio:       user.Bio,
      }
      allUsers = append(allUsers, &owner)
      sinceVal = int64(*owner.ID)
    }
    if sinceVal >= end {
      return allUsers, nil
    }
    if len(users) == 0 {
      break
    }
    opt.Since = int64(sinceVal)

    rateLimit, _, _ := client.RateLimits(ctx)
    //Limit:5000, Remaining:2951, Reset:github.Timestamp{2018-10-24 21:00:34 -0500 CDT
    // rateLimit.Core.Remaining = 0
    fmt.Printf("Core: %s\n", rateLimit.Core.String())
    if rateLimit.Core.Remaining == 0 {
      //Sleep until the time reset
      waiting := true
      t, _ := ParseTime(rateLimit.Core.Reset.String())
      // t, _:= ParseTime("2018-10-25 17:20:28 -0500 CDT")
      fmt.Printf("T is %s\n", t)
      //t, _ := time.Parse(time.RFC3339, "2018-10-24 21:00:34 -0500 CDT") //rateLimit.Core.Reset.String())
      for waiting == true{
        current := time.Now()
        fmt.Printf("Now is %s\n", current)
        diff := t.Sub(current)
        if diff.Seconds() >= 0  {
          fmt.Printf("%d Sleeping\n", int(diff.Seconds()))
          time.Sleep(10 * time.Second)
        } else{
          waiting = false
          fmt.Printf("Resuming! %d\n", int(diff.Seconds()))
        }
      }
    }
  }

  return allUsers, nil
}
