package main

import (
	"fmt"
	"net/http"
	"strconv"

	"earapi/tilecalc"

	"github.com/gin-gonic/gin"
)

func tilecalcArrangeHandler(c *gin.Context) {
	width, height, err := parseTileSizeQuery(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	count, err := strconv.Atoi(c.DefaultQuery("count", "0"))
	if err != nil || count <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "count must be a positive integer"})
		return
	}

	opts, err := parseTilecalcOptions(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	result, err := tilecalc.Arrange(width, height, count, opts)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "", "data": result})
}

func tilecalcCoverageHandler(c *gin.Context) {
	width, height, err := parseTileSizeQuery(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	spaceW, spaceH, err := parseSpaceQuery(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	opts, err := parseTilecalcOptions(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	result, err := tilecalc.Coverage(width, height, spaceW, spaceH, opts)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "", "data": result})
}

func parseTileSizeQuery(c *gin.Context) (int, int, error) {
	sizeStr := c.Query("size")
	if sizeStr != "" {
		return tilecalc.ParseDimensions(sizeStr)
	}

	width, _ := strconv.Atoi(c.DefaultQuery("width", "0"))
	height, _ := strconv.Atoi(c.DefaultQuery("height", "0"))
	return tilecalc.NormalizeTileSize(width, height)
}

func parseSpaceQuery(c *gin.Context) (int, int, error) {
	spaceStr := c.Query("space")
	if spaceStr == "" {
		return 0, 0, fmt.Errorf("space is required (e.g. space=300x130)")
	}
	w, h, err := tilecalc.ParseDimensions(spaceStr)
	if err != nil {
		return 0, 0, err
	}
	if w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("space dimensions must be positive")
	}
	return w, h, nil
}

func parseTilecalcOptions(c *gin.Context) (tilecalc.Options, error) {
	opts := tilecalc.Options{
		ToMeters:               queryBool(c, "meter"),
		ToInches:               queryBool(c, "inches"),
		Graph:                  queryBool(c, "graph"),
		SingleDimensionPattern: queryBool(c, "singledimensionpattern"),
	}

	if v := c.Query("minsplit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return opts, err
		}
		opts.MinSplit = n
		opts.MinSplitSet = true
	}
	if v := c.Query("maxsplit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return opts, err
		}
		opts.MaxSplit = n
		opts.MaxSplitSet = true
	}
	return opts, nil
}

func queryBool(c *gin.Context, name string) bool {
	v := c.Query(name)
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		// treat presence of bare flag-like values as true (e.g. graph=1, graph=true)
		return v == "1" || v == "yes"
	}
	return b
}
