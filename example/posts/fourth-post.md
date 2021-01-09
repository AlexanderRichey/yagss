---
title: Hello world!
date: 2021-01-04
---
This is the *newest* post. It should be on the **first** page.

Here's some code:

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
