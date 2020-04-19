package octree

import (
	"fmt"
	"github.com/The-Tensox/protometry"
)

// FIXME
var (
	CAPACITY = 5
)

// Node ...
type Node struct {
	objects  []Object
	region   protometry.Box
	children *[8]Node
}

// Insert ...
func (n *Node) insert(object Object) bool {
	// Object Bounds doesn't fit in node region => return false
	if !object.Bounds.Fit(n.region) {
		return false
	}

	// Number of objects < CAPACITY and children is nil => add in objects
	if len(n.objects) < CAPACITY && n.children == nil {
		n.objects = append(n.objects, object)
		return true
	}

	// Number of objects >= CAPACITY and children is nil => create children,
	// try to move all objects in children
	// and try to add in children otherwise add in objects
	if len(n.objects) >= CAPACITY && n.children == nil {
		n.split()

		objects := n.objects
		n.objects = []Object{}

		// Move old objects to children
		for i := range objects {
			n.insert(objects[i])
		}

	}

	// Children isn't nil => try to add in children otherwise add in objects
	for i := range n.children {
		if n.children[i].insert(object) {
			return true
		}
	}
	n.objects = append(n.objects, object)
	return true
}

func (n *Node) remove(object Object) bool {
	removedObject := false

	// Object outside Bounds
	if !object.Bounds.Fit(n.region) {
		return false
	}

	for i := 0; i < len(n.objects); i++ {
		// Found it ? delete it and return true
		if n.objects[i].Equal(object) {
			// https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
			n.objects = append(n.objects[:i], n.objects[i+1:]...)
			return true
		}
	}

	if n.children != nil {
		for i := range n.children {
			if n.children[i].remove(object) {
				removedObject = true
				break
			}
		}
	}

	// Successfully removed in children
	if removedObject {
		// Try to merge nodes now that we've removed an item
		n.merge()
	}
	return removedObject
}

func (n *Node) getColliding(bounds protometry.Box) []Object {
	// If current node region entirely fit inside desired Bounds,
	// No need to search somewhere else => return all objects
	if n.region.Fit(bounds) {
		return n.getAllObjects()
	}
	var objects []Object
	// If bounds doesn't intersects with region, no collision here => return empty
	if !n.region.Intersects(bounds) {
		return objects
	}
	// return objects that intersects with bounds and its children's objects
	for _, obj := range n.objects {
		if obj.Bounds.Intersects(bounds) {
			objects = append(objects, obj)
		}
	}
	// No children ? Stop here
	if n.children == nil {
		return objects
	}
	// Get the colliding children
	for _, c := range n.children {
		objects = append(objects, c.getColliding(bounds)...)
	}
	return objects
}

func (n *Node) getAllObjects() []Object {
	var objects []Object
	objects = append(objects, n.objects...)
	if n.children == nil {
		return objects
	}
	for _, c := range n.children {
		objects = append(objects, c.getAllObjects()...)
	}
	return objects
}

/* Merge all children into this node - the opposite of Split.
 * Note: We only have to check one level down since a merge will never happen if the children already have children,
 * since THAT won't happen unless there are already too many objects to merge.
 */
func (n *Node) merge() bool {
	totalObjects := len(n.objects)
	if n.children != nil {
		for _, child := range n.children {
			if child.children != nil {
				// If any of the *children* have children, there are definitely too many to merge,
				// or the child would have been merged already
				return false
			}
			totalObjects += len(child.objects)
		}
	}
	if totalObjects > CAPACITY {
		return false
	}

	// Note: We know children != null or we wouldn't be merging
	for i := range n.children {
		curChild := n.children[i]
		numObjects := len(curChild.objects)
		for j := numObjects - 1; j >= 0; j-- {
			curObj := curChild.objects[j]
			n.objects = append(n.objects, curObj)
		}
	}
	// Remove the child nodes (and the objects in them - they've been added elsewhere now)
	n.children = nil
	return true
}

func (n *Node) move(object *Object, newBounds ...float64) bool {
	// Incorrect dimensions
	if (len(newBounds) != 3 && len(newBounds) != 6) || !n.remove(*object) {
		return false
	}
	if len(newBounds) == 3 {
		object.Bounds = *protometry.NewBoxOfSize(*protometry.NewVectorN(newBounds...), object.Bounds.Extents.Get(0)*2)
	} else { // Dimensions = 6
		object.Bounds = *protometry.NewBoxMinMax(newBounds...)
	}
	return n.insert(*object)
}

// Splits the Node into eight children.
func (n *Node) split() {
	subBoxes := n.region.Split()
	n.children = &[8]Node{}
	for i := range subBoxes {
		n.children[i] = Node{region: *subBoxes[i]}
	}
}


/* * * * * * * * * * * * * * * * * Debugging * * * * * * * * * * * * * * * * */
func (n *Node) getNodes() []Node {
	var nodes []Node
	nodes = append(nodes, *n)
	if n.children != nil {
		for _, c := range n.children {
			nodes = append(nodes, c)
			nodes = append(nodes, c.getNodes()...)
		}
	}
	return nodes
}

// GetRegion is used for debugging visualisation outside octree package
func (n *Node) GetRegion() protometry.Box {
	return n.region
}

func (n *Node) getHeight() int {
	if n.children == nil {
		return 1
	}
	max := 0
	for _, c := range n.children {
		h := c.getHeight()
		if h > max {
			max = h
		}
	}
	return max + 1
}

func (n *Node) getNumberOfNodes() int {
	if n.children == nil {
		return 1
	}
	sum := len(n.children)
	for _, c := range n.children {
		n := c.getNumberOfNodes()
		sum += n
	}
	return sum
}

func (n *Node) getNumberOfObjects() int {
	if n.children == nil {
		return len(n.objects)
	}
	sum := len(n.objects)
	for _, c := range n.children {
		n := c.getNumberOfObjects()
		sum += n
	}
	return sum
}

func (n *Node) toString(verbose bool) string {
	var s string
	s = ",\nobjects: [\n"
	if verbose {
		for _, o := range n.objects {
			s += fmt.Sprintf("%v,\n", o.ToString())
		}
	} else {
		s += fmt.Sprintf("%v objects,\n", len(n.objects))
	}
	s += "]\n,children: [\n"
	if verbose {
		if n.children != nil {
			for _, c := range n.children {
				s += fmt.Sprintf( "%v,\n", c.toString(verbose))
			}
		}
	} else {
		s += fmt.Sprintf("%v children,\n", len(n.children))
	}
	s += "],\n"
	return s
}