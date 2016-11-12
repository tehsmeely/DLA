# DLA
Diffusion Limited Aggregation in Go


Diffusion limited aggregation is a simulation of particles clumping together when movement is based on random linear movement (in a 2D plane)

This is typically processed with an NxM grid with a particle in the centre. Particles are sent on a random walk from a random edge, and stop once they touch (attempt to move onto) existing static particles
To speed this up, we can run the particle walks in different threads or, in this case, goroutines.

This implementation also contains control of command line control for grid size. There is also trhe ability to control the weighting of the discrete probabbility distribution which controls which side a particle spawns on and which direction it moves
this is done by importing the [discreteDistribution](https://github.com/tehsmeely/discreteDistribution) package and giving it a slice of values which add up to 100.
By playing with different start and move discrete probability distributions, the resultant outcome can be 
The different outcomes of playing with different start and move discrete probability distributions can be studdied by comparing the PNG images exported after a run.



