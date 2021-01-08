---
title: Getting tired now...
date: 2021-01-03
---
Can I keep these posts going? Here's some code:

```javascript
console.log("hello!")
```

```python
def qsort(arr, cb=lambda a, b: a < b):
    arr_len = len(arr)
    if arr_len == 0:
        return []
    elif arr_len == 1:
        return arr

    middle = int(arr_len / 2)
    pivot = arr.pop(middle)

    left = []
    right = []
    while len(arr):
        val = arr.pop()
        if cb(val, pivot):
            left.append(val)
        else:
            right.append(val)

    return qsort(left, cb) + [pivot] + qsort(right, cb)
```