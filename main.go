package main

import (
	"Airttp/modules"
	"net/rpc"
	"net"
	"log"
	"fmt"
	"os/exec"
	"os"
	"io/ioutil"
	"strings"
	"Airttp/http"
	"strconv"
)

type Http int

func main() {
	http := new(Http)

	server := rpc.NewServer()
	server.RegisterName("Http", http)

	l, e := net.Listen("tcp", ":5005")
	if e != nil {
		log.Fatal("listen error:", e)
	}

	fmt.Println("server start")
	server.Accept(l)
}

func (t *Http) Module(params modules.ModuleParams, result *modules.ModuleParams) error {
	fmt.Println("New Request")
	result.Copy(params)
	binary, err := exec.LookPath("./php/php-cgi.exe")
	if err != nil {
		return err
	}

	cmd := exec.Command(binary)
	env := os.Environ()
	SetEnv(env, params)
	//env = append(env, fmt.Sprintf("MESSAGE_ID=%s", messageId))
	cmd.Env = env

	cmdOut, _ := cmd.StdoutPipe()
	cmdIn, _ := cmd.StdinPipe()
	//cmdErr, _ := cmd.StderrPipe()

	startErr := cmd.Start()
	if startErr != nil {
		return startErr
	}

	cmdIn.Write(params.Res.Body)
	stdOutput, _ := ioutil.ReadAll(cmdOut)

	var lines = strings.Split(string(stdOutput), "\r\n")
	//set headers until you get a blank line...
	for l := 0 ; l < len(lines) ; l++ {
		if (lines[l] == "") {
			if (result.Res.Headers["Content-Length"] == "") {
				result.Res.Headers["Transfer-Encoding"] = "chunked"
			}
			result.Res.Code = http.Values["OK"].Code
			result.Res.Message = http.Values["OK"].Message
			lines = lines[l+1:]
			result.Res.Body = []byte(strings.Join(lines, "\r\n"))
			result.Res.Headers["Content-Length"] = strconv.Itoa(len(result.Res.Body))
			break;
		} else {
			header := strings.Split(lines[l], ":")
			result.Res.Headers[header[0]] = header[1]
		}
	}

	result.Res.Draw()

	err = cmd.Wait()
	return err
}

func SetEnv(env []string, params modules.ModuleParams) {
	env = append(env, "SERVER_SOFTWARE=Airttp")
	env = append(env, "SERVER_PROTOCOL=HTTP/1.1")
	env = append(env, "GATEWAY_INTERFACE=CGI/1.1")
	host, _ := os.Hostname()
	env = append(env, fmt.Sprintf("SERVER_NAME=%s", host))
	env = append(env, "REDIRECT_STATUS_ENV=0")

	env = append(env, fmt.Sprintf("SCRIPT_NAME=%s", params.Req.Uri))
	//reqEnv['PATH_INFO'] = path.normalize(reqEnv['DOCUMENT_ROOT']+reqdata.pathname);
	//reqEnv['PATH_TRANSLATED'] = path.normalize(reqEnv['DOCUMENT_ROOT']+reqdata.pathname);
	query := ""
	for param_key, param_value := range params.Req.Params {
		query = query + "&" + param_key + "=" + param_value
	}
	if (len(query) > 1) {
		query = query[1:]
	}

	env = append(env, fmt.Sprintf("REQUEST_METHOD=%s", params.Req.Method))
	for header_key, header_value := range params.Req.Headers {
		header_key = strings.ToUpper(header_key)
		if (header_key != "CONTENT_LENGTH" && header_key != "CONTENT_TYPE" && header_key != "AUTH_TYPE") {
			header_key = "HTTP_" + header_key
		}
		env = append(env, fmt.Sprintf("%s=%s", strings.Replace(header_key, "-", "_", -1), header_value))
	}
}