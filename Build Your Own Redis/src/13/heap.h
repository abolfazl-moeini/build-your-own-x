#pragma once

#include <stddef.h>
#include <stdint.h>


struct HeapItem {
    uint64_t val = 0;
    size_t *ref = NULL;
};

void heap_update(HeapItem *a, size_t pos, size_t len);
