## File Index

This is the central map that holds the information for each file that exists on disk.
Only fancy part here is (1) it handles deleting the unwanted file on collisions and (2) it has locking.

One could argue (1) does not belong in here, which is probably correct. But its convienient since we want locking on it _anyway_ and we have those exact locks here soooo.. :shrug:.
