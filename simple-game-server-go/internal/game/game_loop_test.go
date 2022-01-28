package game

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)
type (
	backfillTicket struct {
		Id string
		Connection string
		Attributes map[string]float64
	}
)

func Test_approveBackfillTicket(t *testing.T) {
	l := logrus.NewEntry(logrus.New())
	p := path.Join(t.TempDir(), "config.json")

	require.NoError(t, ioutil.WriteFile(p, []byte(`{"allocatedUUID": "77c31f84-b890-48e8-be08-5db9a551bba3"}`), 0600))

	payloadProxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"token":"eyJhbGciOiJSUzI1NiIsImtpZCI6IjAwOWFkOGYzYWJhN2U4NjRkNTg5NTVmNzYwMWY1YTgzNDg2OWJjNTMiLCJ0eXAiOiJKV1QifQ.eyJlbnZpcm9ubWVudF9pZCI6ImJiNjc5ZWMxLTM3ZmItNDZjNi1iMmZjLWNkNDk4NzJlMmMxYSIsImV4cCI6MTY3NDg1NDEzNiwiaWF0IjoxNjQzMzE4MTM2LCJwcm9qZWN0X2d1aWQiOiJlODBlMmZmMS0zZmFhLTRhOTQtOWUyZC1hMDIxMDdhZTJhODMifQ.FejrCFVs351JQmt_QYUGypG6ECy8c2N2WDFu2a7Ww85MvUWXpdB6KRnRdryKIGTNqNrRhP1wHLQZDYtCGZGc36mBoJ3Kz_1yONp3MDmC92cHWP-9duoB5otrkD66TigtIcXruKdD65vBehFHod2gYvAwhnGa0GWJV4TLR927KiFC_O4mkxIAyTYued3rsFRgCXwlePY2kglOcpCaa8r_86hta4QYbZRmdfTu9ZNeW6K92t8cMoUF_01Re7Gq4gZ-UwEi9IQ9E1ltITyfkY6ksmoURGEZKNuicRrzSTAzUpv460YGCJOZSbbA7ua8DR4qcTgZKDpWUN1LEJoYkuovJcAgj_5svOgdAcPAnmwtkpQQsJx1SSwy9ODFgGozis8k3jxbj_nyd-7zve5KG7l6nNbpnQvG8DIJTIGAl-pQQ_lVvhBlcdeaUeiu4zx5DbijEgqiEXGeTEWZegCMDET_4kyEN-Bs8Bzu4wH_w7MPMQANWuQnB5P-Y4t_wKSLLgOUF5yEZnDm5cVOojnIbYCaGOC5IVj8o4ki2vuff92mAdKWOWIYV-9pg24XDlgss6csGw_8vVO-5p9fUHI4d0nRsIB_YeblNrVEcJeiVtVFA_yzx_v9K8AJyt_xZUhsJ3N85E9ftIP5NuHIL0sNxwl7m6dzHQ9XwiQJ_pZU4QFzIJI","error":""}`)
	}))
	defer payloadProxyServer.Close()

	mmBackfillServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{ "Id": "77c31f84-b890-48e8-be08-5db9a551bba3", "Connection": "127.0.0.1:9555", "Attributes": { "att1": 100.0 } }`)
	}))
	defer mmBackfillServer.Close()

	g, err := New(l, p, 9000, 9001, &http.Client{Timeout: time.Duration(1) * time.Second}, payloadProxyServer.URL, mmBackfillServer.URL)
	require.NoError(t, err)
	require.NotNil(t, g)

	resp, err := g.approveBackfillTicket()
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotNil(t, bodyBytes)

	var ticket backfillTicket
	err = json.Unmarshal(bodyBytes, &ticket)
	require.NoError(t, err)
	require.NotNil(t, ticket)

	require.Equal(t, "77c31f84-b890-48e8-be08-5db9a551bba3", ticket.Id)
	require.Equal(t, "127.0.0.1:9555", ticket.Connection)
	require.Equal(t, 1, len(ticket.Attributes))
	require.Equal(t, 100.0, ticket.Attributes["att1"])

	close(g.done)
}
