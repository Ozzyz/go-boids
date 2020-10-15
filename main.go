package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"syscall/js"
	"time"
)

/*
BOIDS
This program implements a boid flocking pattern.
The main algorithms employed are taken from here: http://www.vergenet.net/~conrad/boids/pseudocode.html
There are still a lot of improvements that can be done. For instance, adding acceleration, attractors,
and the ability to add more boids on click.
*/

// Config parameters
const windowWidth, windowHeight = 1280, 760
const numBoids = 50
const nearbyDistance = 25
const numNeighbours = 7
const boidSize = 5

// Netlight colors:
// FFACAC
// 2C3F6B
// 7473BD
var boidColor = "#7473BD"

/*================== Structs and helper methods ================== */

type Vec2 struct {
	// This is a general struct used to represent both velocity and position for a boid
	X float64
	Y float64
}

// Add (addition) method for Vec2
func (a Vec2) Add(b Vec2) Vec2 {
	a.X += b.X
	a.Y += b.Y
	return a
}

func (a Vec2) Subtract(b Vec2) Vec2 {
	// Performs the vector operation a-b
	a.X -= b.X
	a.Y -= b.Y
	return a
}

func (a Vec2) Divide(b float64) Vec2 {
	a.X /= b
	a.Y /= b
	return a
}

func (a Vec2) Multiply(b float64) Vec2 {
	a.X *= b
	a.Y *= b
	return a
}

func (a Vec2) Dist(b Vec2) float64 {
	// Calculates the Euclidean distance between the two vectors
	X := a.X - b.X
	Y := a.Y - b.Y
	return math.Sqrt(X*X + Y*Y)
}

func (a Vec2) Length() float64 {
	// Returns the length of the vector - which is the same as distance from origo
	return a.Dist(Vec2{0, 0})
}

func (boid Boid) nearestNeighbours(boids []*Boid) []Boid {
	// Returns an array of all the neighbours of the boid
	neighbours := make([]Boid, len(boids))
	for _, otherBoid := range boids {
		neighbours = append(neighbours, *otherBoid)
	}
	sort.SliceStable(neighbours, func(i, j int) bool {
		return boid.position.Dist(neighbours[i].position) < boid.position.Dist(neighbours[j].position)
	})
	return neighbours[0:numNeighbours]
}

type Boid struct {
	// The struct representing one bird (boid)
	position Vec2
	velocity Vec2
}

func main() {

	fmt.Println("Configuring canvas!")
	initCanvas(windowHeight, windowWidth)
	fmt.Println("Initializing boids")
	boids := initializeBoids(numBoids)
	fmt.Printf("Initialized %d boids", len(boids))
	for {
		drawBoids(boids)
		updateBoidPositions(boids)
		time.Sleep(5 * time.Millisecond)
	}
}

func initCanvas(height int, width int) {
	var canvas js.Value = js.Global().
		Get("document").
		Call("getElementById", "canvas")

	// Config height and width
	canvas.Set("height", height)
	canvas.Set("width", width)
	// Reset canvas
	var context js.Value = canvas.Call("getContext", "2d")
	context.Call("clearRect", 0, 0, width, height)
}

func drawBoids(boids []*Boid) {
	// Draw each boid within bounds on the canvas.
	// The direction is based on the velocity vector
	var canvas js.Value = js.Global().
		Get("document").
		Call("getElementById", "canvas")

	var context js.Value = canvas.Call("getContext", "2d")
	// Clear canvas from previous drawings
	context.Call("clearRect", 0, 0, windowWidth, windowHeight)
	for _, boid := range boids {
		drawSingleBoid(context, boid)
	}
}

func drawSingleBoid(context js.Value, boid *Boid) {
	var boidPos = boid.position
	var boidDir = boid.velocity

	// Save coordinate system before we mess with it
	context.Call("save")
	// Translate coordinate system to the middle of where we want to draw our object
	context.Call("translate", boidPos.X, boidPos.Y)
	// Rotate triangle to direction of velocity of boid
	var radians = math.Atan2(boidDir.Y, boidDir.X) + math.Pi/2
	// Perform the rotation of the coordinate system
	context.Call("rotate", radians)

	// Draw the object - this is just a basic triangle
	context.Call("beginPath")
	context.Call("moveTo", 0, 0)
	context.Call("lineTo", -boidSize, boidSize*4)
	context.Call("lineTo", boidSize, boidSize*4)
	context.Set("fillStyle", boidColor)
	context.Call("fill")
	// Revert back to the original coordinate system
	context.Call("restore")
}
func updateBoidPositions(boids []*Boid) {
	// Boids are updated based on three rules
	// 1) Boids fly towards the centre of mass of all boids
	// 2) Boids want to keep a small distance to other nearby boids
	// 3) Boids want to keep the same velocity as other nearby boids
	// These rules do NOT modify the position of the boid, only tells them how to adjust the velocity
	// based on the rule.
	// Returns the modified list
	for i, boid := range boids {
		var currentBoid = boids[i]
		// TODO: This does not have to be calculated for every single boid,
		// since if a is a neighbour of b, then b is a neighbour of a
		var neighbours = currentBoid.nearestNeighbours(boids)

		var v1 = centreOfMassRule(neighbours, currentBoid)
		var v2 = nearbyRule(neighbours, currentBoid)
		var v3 = velocityRule(neighbours, currentBoid)

		// Add all velocities to current velocity
		currentBoid.velocity = currentBoid.velocity.Add(v1).Add(v2).Add(v3)
		// Cap velocity so that we don't get insane speed
		limitVelocity(currentBoid)
		// Update position - just velocity of one timestep
		currentBoid.position = currentBoid.position.Add(currentBoid.velocity)
		stayInWindow(boid)
	}

}

func centreOfMassRule(boids []Boid, currentBoid *Boid) Vec2 {
	// Calculates the centre of mass of the entire flock and returns a vec2 that
	// describes the position of this flock, with a discontinuation factor to about 1% of the centre of mass
	var total = Vec2{0, 0}
	for _, boid := range boids {
		total = total.Add(boid.position)
	}
	// Divide by total boids to normalize to average position

	var centre = total.Divide(float64(len(boids))).Subtract(currentBoid.position).Divide(float64(100))
	//fmt.Printf("New direction for centreOfMassRule: (%d, %d)\n", total.X, total.Y)
	return centre
}

func nearbyRule(boids []Boid, currentBoid *Boid) Vec2 {
	// We want nearby boids to keep a bit of distance from eachother
	// Therefore, for every boid that is near another boid, give a nudge in the opposite direction
	// Input:
	//	boids: A list of all boids
	//  curBoid: The index of the boid we are currently evaluating

	var Direction = Vec2{0, 0}
	for _, boid := range boids {
		if boid.position.Dist(currentBoid.position) < nearbyDistance {
			var negDirection = currentBoid.position.Subtract(boid.position)
			Direction = Direction.Add(negDirection)
		} else {
			//fmt.Printf("Distance between boid %d and %d is %f\n", i, curBoid, boids[i].position.Dist(boids[curBoid].position))
		}

	}
	//fmt.Printf("NearbyRule returned new direction (%f, %f)\n", Direction.X, Direction.Y)
	return Direction
}

func velocityRule(boids []Boid, curBoid *Boid) Vec2 {
	// Average the velocities of each boid, then add this to the current velocity of the boid
	var total = Vec2{0, 0}
	for i := 0; i < len(boids); i++ {
		total = total.Add(boids[i].velocity)
	}
	// Divide by total boids to normalize to average position
	total = total.Divide(float64(len(boids)))
	var newDirection = total.Subtract(curBoid.velocity).Divide(20)
	//fmt.Printf("New direction for velocityRule: (%f, %f)\n", newDirection.X, newDirection.Y)
	return newDirection
}

func initializeBoids(numBoids int) []*Boid {
	// Initialize a set of boid objects with random starting positions
	// Returns a list of boid objects
	boids := make([]*Boid, numBoids)
	for i := 0; i < numBoids; i++ {
		boid := randomBoid()
		boids[i] = &boid
	}
	return boids
}

func limitVelocity(boid *Boid) {
	// Limits the velocity but does not change the direction of the boid
	var velocityMagnitude = 5.0

	var vecLength = boid.velocity.Length()
	if vecLength > velocityMagnitude {
		boid.velocity = boid.velocity.Divide(vecLength).Multiply(velocityMagnitude)
	}
}

func randomBoid() Boid {
	// Returns a Boid object with random starting position
	var initX, initY = float64(rand.Intn(windowWidth)), float64(rand.Intn(windowHeight))
	var initialPosition = Vec2{initX, initY}
	var vInitX, vInitY = float64(rand.Intn(10) - 5), float64(rand.Intn(10) - 5)
	var initialVelocity = Vec2{vInitX, vInitY}
	boid := Boid{initialPosition, initialVelocity}
	return boid
}

// If boid goes outside window, it pops out on the other side
func stayInWindow(boid *Boid) {
	if boid.position.X < 0 {
		boid.position.X = windowWidth + boid.position.X
	} else if boid.position.X > windowWidth {
		boid.position.X = windowWidth - boid.position.X
	}
	if boid.position.Y < 0 {
		boid.position.Y = windowHeight + boid.position.Y
	} else if boid.position.Y > windowHeight {
		boid.position.Y = windowHeight - boid.position.Y
	}
}
