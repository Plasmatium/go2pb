package main

type CyclicDetector struct {
	// importsMap: 
	// key: messageName
	// value: importPath
	importsMap map[string]string

	// importsMapReversed:
	// key: importPath
	// value: messageNameList
	importsMapReversed map[string][]string

	// messagesMap:
	// key: messageName
	// value: protoMessage
	messagesMap map[string]*ProtoMessage

	dependencyGraph []*node

	visitedImports map[string]struct{}
}

func NewCyclicDetector(
	importsMap map[string]string,
	importsMapReversed map[string][]string,
	messagesMap map[string]*ProtoMessage,
) *CyclicDetector {
	return &CyclicDetector{
		importsMap:         importsMap,
		importsMapReversed: importsMapReversed,
		messagesMap:        messagesMap,
	}
}

func (cd *CyclicDetector) append(importPath string) {
	if _, visited := cd.visitedImports[importPath]; visited {
		return
	} else {
		cd.visitedImports[importPath] = struct{}{}
	}
	root := &node{
		name: importPath,
	}
	cd.dependencyGraph = append(cd.dependencyGraph, root)
	msgs := cd.importsMapReversed[importPath]
	for _, msg := range msgs {
		msgImportPath, found := cd.importsMap[msg]
		if !found {
			continue
		}
		root.addNext(&node{
			name: msgImportPath,
		})
		cd.visitedImports[msgImportPath] = struct{}{}
	}
}

// DetectCyclicPaths detects cyclic paths in the import graph.
func (cd CyclicDetector) DetectCyclicPaths(pathName string) (cyclicPaths []string, found bool) {

	return
}

// MakeMergeMap creates a map that store a path if should merged into another path.
func (cd CyclicDetector) MakeMergeMap() (mergeMap map[string]string) {
	return
}


type node struct {
	name string
	visited bool
	inPath bool
	next *node
}

func (n *node) addNext(next *node) {
	n.next = next
}