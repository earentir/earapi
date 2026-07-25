package main

import (
	"net/http"
	"strconv"
	"time"

	"earapi/dmt"

	"github.com/gin-gonic/gin"
)

func dmtTimestampHandler(c *gin.Context) {
	in, err := parseDMTInput(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	result, err := dmt.Convert(in)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "", "data": result})
}

func dmtFormatsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"msg":     "",
		"data":    dmt.Styles,
	})
}

func parseDMTInput(c *gin.Context) (dmt.Input, error) {
	in := dmt.Input{
		DateTime: c.Query("datetime"),
		Style:    c.DefaultQuery("format", c.Query("style")),
		Offset:   c.Query("offset"),
		Complete: queryBool(c, "complete"),
	}

	if v := c.Query("unix"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return in, err
		}
		in.Unix = &n
		return in, nil
	}
	if v := c.Query("epoch"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return in, err
		}
		in.Unix = &n
		return in, nil
	}

	if in.DateTime != "" {
		return in, nil
	}

	hasAny := false
	for _, key := range []string{"year", "month", "day", "hour", "minute", "second"} {
		if c.Query(key) != "" {
			hasAny = true
			break
		}
	}
	if !hasAny {
		now := time.Now().Unix()
		in.Unix = &now
		return in, nil
	}

	year, err := strconv.Atoi(c.Query("year"))
	if err != nil {
		return in, err
	}
	month, err := strconv.Atoi(c.DefaultQuery("month", "1"))
	if err != nil {
		return in, err
	}
	day, err := strconv.Atoi(c.DefaultQuery("day", "1"))
	if err != nil {
		return in, err
	}
	hour, _ := strconv.Atoi(c.DefaultQuery("hour", "0"))
	minute, _ := strconv.Atoi(c.DefaultQuery("minute", "0"))
	second, _ := strconv.Atoi(c.DefaultQuery("second", "0"))

	in.Year = year
	in.Month = month
	in.Day = day
	in.Hour = hour
	in.Minute = minute
	in.Second = second
	in.HasComponents = true
	return in, nil
}
