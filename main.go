package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-ldap/ldap"
	"io/ioutil"
	"k8s.io/api/authentication/v1"
	"log"
	"net/http"
	"os"
	"strings"
	"regexp"
)

var ldapURL string

func main() {
	ldapURL = "ldap://" + os.Args[1]
	log.Printf("Using LDAP directory %s\n", ldapURL)
	log.Println("Listening on port 443 for requests...")
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServeTLS(":443", os.Args[3], os.Args[2], nil))
}

func handler(w http.ResponseWriter, r *http.Request) {

	// Read body of POST request
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeError(w, err)
		return
	}
	log.Printf("Receiving: %s\n", string(b))

	// Unmarshal JSON from POST request to TokenReview object
	// TokenReview: https://github.com/kubernetes/api/blob/master/authentication/v1/types.go
	var tr v1.TokenReview
	err = json.Unmarshal(b, &tr)
	if err != nil {
		writeError(w, err)
		return
	}

	// Extract username and password from the token in the TokenReview object
	s := strings.SplitN(tr.Spec.Token, ":", 2)
	if len(s) != 2 {
		writeError(w, fmt.Errorf("badly formatted token: %s", tr.Spec.Token))
		return
	}
	username, password := s[0], s[1]

	// Make LDAP Search request with extracted username and password
	userInfo, err := ldapSearch(username, password)
	if err != nil {
		writeError(w, fmt.Errorf("failed LDAP Search request: %v", err))
		return
	}

	// Set status of TokenReview object
	if userInfo == nil {
		tr.Status.Authenticated = false
	} else {
		tr.Status.Authenticated = true
		tr.Status.User = *userInfo
	}

	// Marshal the TokenReview to JSON and send it back
	b, err = json.Marshal(tr)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Write(b)
	log.Printf("Returning: %s\n", string(b))
}

func writeError(w http.ResponseWriter, err error) {
	err = fmt.Errorf("Error: %v", err)
	w.WriteHeader(http.StatusInternalServerError) // 500
	fmt.Fprintln(w, err)
	log.Println(err)
}

func ldapSearch(username, password string) (*v1.UserInfo, error) {

	// Connect to LDAP directory
	l, err := ldap.DialURL(ldapURL)
	if err != nil {
		return nil, err
	}
	defer l.Close()

	// Authenticate as LDAP user
	user :=  fmt.Sprintf("%s@KUBER.NET", username)	
	uPass := fmt.Sprintf("%s", password)
	err = l.Bind(user, uPass)
	if err != nil {
		return nil, err
	}

	// Execute LDAP Search request
	searchRequest := ldap.NewSearchRequest(
		"cn=Users,dc=kuber,dc=net",  // Search base
		ldap.ScopeWholeSubtree, // Search scope
		ldap.NeverDerefAliases, // Dereference aliases
		0,                      // Size limit (0 = no limit)
		0,                      // Time limit (0 = no limit)
		false,                  // Types only
		fmt.Sprintf("(&(sAMAccountName=%s))", username), // Filter
		nil, // Attributes (nil = all user attributes)
		nil, // Additional 'Controls'
	)
	result, err := l.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	// If LDAP Search produced a result, return UserInfo, otherwise, return nil
	if len(result.Entries) == 0 {
		return nil, nil
	} else {
	        groups := result.Entries[0].GetAttributeValues("memberOf")
      	        for i := 0; i < len(groups); i++ {
       		 	groups[i] = regexp.MustCompile("CN=").ReplaceAllString(
                	regexp.MustCompile(",").ReplaceAllString(
                	regexp.MustCompile("CN=([A-z0-9À-ú\t\n\v\f\r _-]+),").FindString(string(groups[i])), ""),"")
              	}
		return &v1.UserInfo{
			Username: username,
			UID:      username,
			Groups:   groups,
		}, nil
	}
}
