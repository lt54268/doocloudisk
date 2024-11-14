package service

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

type User struct {
	Userid     int      `json:"userid"`
	Identity   []string `json:"identity"`   // Assuming identity is a slice of strings
	Department []string `json:"department"` // Assuming department is a slice of strings
	Az         string   `json:"az"`
	Email      string   `json:"email"`
	Nickname   string   `json:"nickname"`
	Userimg    string   `json:"userimg"`
	LoginNum   int      `json:"login_num"`
	Changepass int      `json:"changepass"`
	LastIp     string   `json:"last_ip"`
	LastAt     string   `json:"last_at"`
	LineIp     string   `json:"line_ip"`
	LineAt     string   `json:"line_at"`
	CreatedIp  string   `json:"created_ip"`
}

func GetUserID(token []byte) (user_id int, err error) {
	c, err := client.NewClient()
	if err != nil {
		return
	}
	req := &protocol.Request{}
	res := &protocol.Response{}
	req.SetMethod(consts.MethodGet)
	req.Header.Set("TOKEN", string(token))
	ip := os.Getenv("NGINX_URL")
	if ip == "" {
		ip = "nginx"
	}
	req.SetRequestURI(fmt.Sprintf("http://%s/api/users/info", ip))
	err = c.Do(context.Background(), req, res)
	if err != nil {
		return
	}
	fmt.Printf("%v\n", string(res.Body()))
	return 1, nil
}
