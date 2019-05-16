/*
  Exergy BASIC

  Author: Peter Hague

  TODO:
    - Fix the page tag in the URL (proper encoding)
    - Make sure that page requests are idempotent
    - Add the LET command, and variables
*/

package main

import (
    "fmt"
    "log"
    "net/http"
    "strings"
    "strconv"
    "text/template"
    "io/ioutil"
  	"crypto/md5"
)


var operatorPrec =  "^ / * - + ("
var outputBuffer =  "Exergy BASIC<br /><br />V0.1<br /><br />"
var lineCount = 0

type outputFrame struct {
  Textcontent string
  Ptag [16]byte
}

type ErrSyntax struct {
  message string
}

func NewErrSyntax(message string) *ErrSyntax {
  return &ErrSyntax{
    message: message,
  }
}

func (e *ErrSyntax) Error() string {
  return e.message
}

// Convert expression to RP using shunting yard algorithm
// Currently, all operators are left associative
func exprParse(expr string) (string, error) {
  var reversePolish string
  var opStack []string

  for i:=0;i<len(expr);i++ {
    if x := strings.Index(operatorPrec,string(expr[i])); x>-1 {
      if expr[i] != '(' {
        for len(opStack)>0 {
          if y := strings.Index(operatorPrec, opStack[len(opStack)-1]); y<=x {
            reversePolish += ","+opStack[len(opStack)-1]
            opStack = opStack[:len(opStack)-1]
          } else { break }
        }
        reversePolish += ","
      }
      opStack = append(opStack,string(expr[i]))
    } else {
      if expr[i] == ')' {
        for len(opStack)>0 && opStack[len(opStack)-1]!="(" {
          reversePolish += ","+opStack[len(opStack)-1]
          opStack = opStack[:len(opStack)-1]
        }
        if len(opStack)==0 { return "", NewErrSyntax("Mismatched ()")}
        opStack = opStack[:len(opStack)-1]
      } else {
        reversePolish += string(expr[i])
      }
    }
  }
  for len(opStack)>0 {
    if opStack[len(opStack)-1] == "(" {
      return "", NewErrSyntax("Mistmatched ()")
    }
    reversePolish += ","+opStack[len(opStack)-1]
    opStack = opStack[:len(opStack)-1]
  }
  return reversePolish, nil
}

func evaluate(expression string) (float64, error) {
  stack := make([]float64,10)
  tokens := strings.Split(expression, ",")
  for _,t := range(tokens) {
    if len(t)==0 { continue }
    if x := strings.Index("0123456789.", string(t[0])); x>-1 {
      val, _ := strconv.ParseFloat(t,64)
      stack = append(stack, val)
    }
    switch t[0] {
      case '+':
        result := stack[len(stack)-2]+stack[len(stack)-1]
        stack = stack[:len(stack)-2]
        stack = append(stack, result)
      case '*':
        result := stack[len(stack)-2]*stack[len(stack)-1]
        stack = stack[:len(stack)-2]
        stack = append(stack, result)
      case '-':
        result := stack[len(stack)-2]-stack[len(stack)-1]
        stack = stack[:len(stack)-2]
        stack = append(stack, result)
      case '/':
        result := stack[len(stack)-2]/stack[len(stack)-1]
        stack = stack[:len(stack)-2]
        stack = append(stack, result)
    }
  }
  return stack[len(stack)-1], nil
}

//Handle the HTTP requests
func handler(w http.ResponseWriter, r *http.Request) {
    statement := r.URL.Query().Get("cmd")

    if statement!="" {
      lineCount += 1
      outputBuffer += fmt.Sprintf("<div class='statement'>%s</div><br />", statement)

      tokens := strings.Split(statement, " ")
      switch tokens[0] {
        case "clear":
          outputBuffer = ""
        case "print":
          rp_expr, err := exprParse(statement)
          result, err := evaluate(rp_expr)
          if (err != nil) {
            outputBuffer += err.Error()+"<br />"
          } else {
            outputBuffer += fmt.Sprintf("%f", result)+"<br />"
          }
        default:
          outputBuffer += "Command not regonised<br />"
      }


    }

    p := md5.Sum([]byte(fmt.Sprintf("exergy%s%f",statement,lineCount)))
    newpage := outputFrame{outputBuffer, p}

    pagetemplate, err := ioutil.ReadFile("mainpage.html")
    tmpl, err := template.New("Output").Parse(string(pagetemplate))

    if (err == nil) {
      tmpl.Execute(w, newpage)
    } else {
      panic(err)
    }
}

func main() {
    http.HandleFunc("/", handler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
