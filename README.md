# classificator

a tool for taking text responses and classifying them by hand. I just made it
for myself but open sourcing it anyway.

It stores everything in a SQLite database.

### how to use it

these instructions should work:

```
$ sqlite3 comments.db < schema.sql
$ go run .
```

Then open http://localhost:8080

### a screenshot

Here's a screenshot of what the main interface looks like:

![Screenshot 2025-02-05 at 2 37 22 PM](https://github.com/user-attachments/assets/31fe3d3c-f4b4-49b0-a768-fb05ad2686ad)
