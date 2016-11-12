package main


import (
	"fmt"
	"sync" 
	"strings"
	"strconv"
	"math/rand"
	"image"
	"image/color"
	"image/png"
	"os"
	"log"
	"time"

	"github.com/urfave/cli"

	"github.com/tehsmeely/discreteDistribution"
)

var (
	OpsLog *log.Logger
	RunLog *log.Logger
)

// PARTICLE
type Particle struct {
	X int
	Y int
	lastX int
	lastY int
	dc *DiffuseConfig
}

func (p *Particle) Init(grid *Grid){
	//pick a size, 0-3
	//pick a point along that side, 0 - sizeX/Y-1
	//side := rand.Intn(4)
	side, _ := discreteDistribution.Generate(p.dc.start, []int{0,1,2,3})
	switch side {
	case 0: //top
		p.X = rand.Intn(grid.sizeX)
		p.Y = 0
	case 1: //right
		p.X = grid.sizeX-1
		p.Y = rand.Intn(grid.sizeY)
	case 2: //bottom
		p.X = rand.Intn(grid.sizeX)
		p.Y = grid.sizeY-1
	case 3: //left
		p.X = 0
		p.Y = rand.Intn(grid.sizeY)
	}
	p.lastX = p.X
	p.lastY = p.Y
}
func (p *Particle) Move(grid *Grid){
	//side := rand.Intn(4)
	//experimental, Left/Right is weighted 2xUp/Down
	side, _ := discreteDistribution.Generate(p.dc.move, []int{0,1,2,3})
	switch side {
	case 0: //up
		if p.Y > 0{
			p.lastY = p.Y
			p.Y--
		}
	case 1: //right
		if p.X < grid.sizeX-1 {
			p.lastX = p.X
			p.X++
		}
	case 2: //down
		if p.Y < grid.sizeY-1 {
			p.lastY = p.Y
			p.Y++
		}
	case 3: //left
		if p.X > 0{
			p.lastX = p.X
			p.X--
		}
	}
}
func (p *Particle) Revert(){
	p.X = p.lastX
	p.Y = p.lastY
}

// DiffuseConfig
type DiffuseConfig struct{
	move 	[]int
	start   []int
}
func (dc *DiffuseConfig) Init(moves, starts string) {
	//trust the strings already, they went through validation
	moveslice := strings.Split(moves, ",")
	move:= make([]int, len(moveslice))
	for i, val := range moveslice {
		intval, _ := strconv.Atoi(val)
		move[i] = intval
	}
	dc.move = move
	startslice := strings.Split(starts, ",")
	start:= make([]int, len(startslice))
	for i, val := range startslice {
		intval, _ := strconv.Atoi(val)
		start[i] = intval
	}
	dc.start = start
}

// GRID
type Grid struct {
	sizeX, sizeY 	int
	grid 			[][]uint8
	finished 		bool
	mux 			sync.Mutex
}

// GRID METHODS START
func (grid *Grid) Init() {
	rows := make([]uint8, grid.sizeX*grid.sizeY)
	for i := range grid.grid {
		grid.grid[i], rows = rows[:grid.sizeX], rows[grid.sizeX:]
	}
}

func (grid *Grid) Print() {
	for j:=0; j < len(grid.grid[0]); j++ {
		for i :=  range grid.grid[j] {
			fmt.Printf("%v ", grid.grid[i][j])
		}
		fmt.Printf("\n")
	}
}

func (grid *Grid) GetAt(x, y int) (uint8, bool) {
	grid.mux.Lock()
	val := grid.grid[x][y]
	finished := grid.finished
	grid.mux.Unlock()
	return val, finished
}

func (grid *Grid) SetAt(x , y int, val uint8) {
	grid.mux.Lock()
	grid.grid[x][y] = val
	grid.mux.Unlock()
}

func (grid *Grid) Place(p *Particle) {
	grid.mux.Lock()
	grid.grid[p.X][p.Y] = 1
	if (p.X == 0) || (p.X == grid.sizeX-1) || (p.Y == 0) || (p.Y == grid.sizeY-1) {
		grid.finished = true
	}
	grid.mux.Unlock()
}
// GRID METHODS END

func main() {
	app := cli.NewApp()
	app.Version = "0.0.1"
	app.Name = "DLA"
	app.Usage = "Diffusion Limited Aggregation"
	app.Flags = []cli.Flag {
		cli.StringFlag{
			Name:        "output, o",
			Value:       "DLA.out.png",
			Usage:       "output image filename",
		},
		cli.StringFlag{
			Name:        "move, m",
			Value:       "25,25,25,25",
			Usage:       "CSV 4-item list for weighting for random movement. Sum must be 100: \"up,right,down,left\"",
		},
		cli.StringFlag{
			Name:        "start, s",
			Value:       "25,25,25,25",
			Usage:       "CSV 4-item list for weighting for random start. Sum must be 100: \"up,right,down,left\"",
		},
		cli.BoolFlag{
			Name:        "verbose, report, r",
			Usage:       "Enable verbose routine reporting",
		},
		cli.BoolFlag{
			Name:        "nooutput, n",
			Usage:       "Do not output resulting image",
		},
	}
	app.Action = customDLA
	app.Run(os.Args) 
}

func validateArgs(c *cli.Context) (bool, string) {
///Validate all args from cli
	if len(c.Args()) != 2{
		return false, "Supply two arguments: <xSize> <ySize> (+options)"
	}
	if _, err := strconv.Atoi(c.Args().Get(0)); err != nil {
		return false, "<xSize> should be an integer"
	}

	if _, err := strconv.Atoi(c.Args().Get(1)); err != nil {
		return false, "<ySize> should be an integer"
	}

	//output
	if !strings.HasSuffix(c.String("output"),".png"){
		return false, "'output' filename must end \".png\""
	} 

	var sum int
	//move
	if mlen:= len(strings.Split(c.String("move"), ",")); mlen < 4 {
		return false, "'move' list too short, should be 4 comma separated"
	} else if mlen > 4 {
		return false, "'move' list too long, should be 4 comma separated"
	}
	//	check move sum is 100
	for _, val := range strings.Split(c.String("move"), ",") {
		ival, err := strconv.Atoi(val)
		if err != nil {
			return false, "'start' list does not contain regular integers"
		}
		sum+=ival
	}
	if sum != 100 {
		return false, fmt.Sprintf("'move' list doest not add up to 100: (Adds up to %v)", sum)
	}
	sum=0
	//start
	if slen:= len(strings.Split(c.String("start"), ",")); slen < 4 {
		return false, "'start' variable too short, should be 4 comma separated"
	} else if slen > 4 {
		return false, "'start' variable too long, should be 4 comma separated"
	} 
	//	check start sum is 100
	for _, val := range strings.Split(c.String("start"), ",") {
		ival, err := strconv.Atoi(val)
		if err != nil {
			return false, "'start' list does not contain regular integers"
		}
		sum+=ival
	}
	if sum != 100 {
		return false, fmt.Sprintf("'start' list doest not add up to 100: (Adds up to %v)", sum)
	}
	//verbose - unneccessary
	//nooutput - unneccessary

	return true, ""
}

func customDLA(c *cli.Context) error {
// Main function, spawn goroutines for diffusing particles
	rand.Seed( time.Now().UTC().UnixNano())
	logFile, err := os.OpenFile("DLA.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	start := time.Now()
	if err != nil {
		fmt.Println("Failed to open log file 'DLA.log' :", err)
		return err
	}
	OpsLog=log.New(logFile, "", log.Ldate|log.Ltime|log.Lshortfile)
	OpsLog.Println("Starting")
	logFile2, err := os.OpenFile("DLArun.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Failed to open log file 'DLArun.log' :", err)
		return err
	}
	RunLog=log.New(logFile2, "", 0)

	ALPHABET := [26]string{"A","B","C","D","E","F","G","H","I","J","K","L","M","N","O","P","Q","R","S","T","U","V","W","X","Y","Z"}
	var dc DiffuseConfig 
	ok, errorText :=  validateArgs(c)
	if !ok{
		fmt.Println(errorText)
		OpsLog.Println("Failed to validate args: ", errorText)
		return nil
	}
	XSize, _ := strconv.Atoi(c.Args().Get(0))
	YSize, _ := strconv.Atoi(c.Args().Get(1))
	dc.Init(c.String("start"), c.String("move"))
	grid := Grid{sizeX: XSize, sizeY: YSize, grid: make([][]uint8, YSize)}
	grid.Init()	



	grid.SetAt(XSize/2, YSize/2, 1)
	fmt.Println("Starting")
	var wg sync.WaitGroup
	for i:=0; i < 10; i++{
		wg.Add(1)
		go diffuse(&grid, &wg, &dc, ALPHABET[i], c.Bool("verbose"))
	}
	wg.Wait()
	fmt.Println()
	var success bool
	elapsed := time.Since(start) // Before export so it doesnt factor in that time
	if !c.Bool("nooutput") {
		success = export(&grid, c.String("output"))
	}	
	if success {
		fmt.Println("Done, exported to", c.String("output"))
		OpsLog.Printf("Complete. Took %v minutes. Exported result with s%v and m%v to %v\n", elapsed.Minutes(), c.String("start"), c.String("move"), c.String("output"))
		RunLog.Printf("{\"SIZE\":[%v,%v],\"MOVE\":\"%v\",\"START\":\"%v\",\"TIME\":{\"SECONDS\":\"%v\", \"MINUTES\":\"%v\"}}\n", XSize, YSize, c.String("start"), c.String("move"), elapsed.Seconds(), elapsed.Minutes())
	} else if !c.Bool("nooutput") {
		fmt.Println("Done, not exported")
		OpsLog.Printf("Complete. Took %v minutes. Result with s%v and m%v was not exported\n", elapsed.Minutes(), c.String("start"), c.String("move"))
		RunLog.Printf("{\"SIZE\":[%v,%v],\"MOVE\":\"%v\",\"START\":\"%v\",\"TIME\":{\"SECONDS\":\"%v\", \"MINUTES\":\"%v\"}}\n", XSize, YSize, c.String("start"), c.String("move"), elapsed.Seconds(), elapsed.Minutes())
	} else {
		fmt.Println("Done. Failed to export to image file", c.String("output"))
		OpsLog.Printf("Exporting file to %v failed. Done (took %v minutes)\n", c.String("output"), elapsed.Minutes())
	}
	return nil
}

func diffuse(grid *Grid, wg *sync.WaitGroup, dc *DiffuseConfig, name string, v bool){
	defer wg.Done()
	p := Particle{dc: dc}
	p.Init(grid)
	for{
		p.Move(grid)
		val, finished := grid.GetAt(p.X, p.Y)
		if finished {
			return
		}
		if val > 0 {
			p.Revert()
			if v {fmt.Printf("%v ", name)}
			grid.Place(&p)
			p.Init(grid)
		}

	}
}


func export(grid *Grid, output string) bool {
	myimage := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{grid.sizeX, grid.sizeY}})
	var c color.RGBA
	for x := 0; x < grid.sizeX; x++ {
		for y := 0; y < grid.sizeY; y++ {
			if set, _ := grid.GetAt(x, y); set > 0 {
				c = color.RGBA{255, 255, 255, 255}
			} else {
				c = color.RGBA{0, 0, 0, 255}
			}
			myimage.Set(x, y, c)
		}
	}

	myfile, err := os.Create(output)
	if err == nil {
		png.Encode(myfile, myimage)
		return true
	} else {
		return false
	}
}