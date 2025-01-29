#include <assert.h>
#include <vector>
#include <map>
#include "heap.cpp"


struct Data {
    size_t heap_idx = -1;
};

struct Container {
    std::vector<HeapItem> heap;
    std::multimap<uint64_t, Data *> map;
};

static void dispose(Container &c) {
    for (auto p : c.map) {
        delete p.second;
    }
}

static void add(Container &c, uint64_t val) {
    Data *d = new Data();
    c.map.insert(std::make_pair(val, d));
    HeapItem item;
    item.ref = &d->heap_idx;
    item.val = val;
    c.heap.push_back(item);
    heap_update(c.heap.data(), c.heap.size() - 1, c.heap.size());
}

static void del(Container &c, uint64_t val) {
    auto it = c.map.find(val);
    assert(it != c.map.end());
    Data *d = it->second;
    assert(c.heap.at(d->heap_idx).val == val);
    assert(c.heap.at(d->heap_idx).ref == &d->heap_idx);
    c.heap[d->heap_idx] = c.heap.back();
    c.heap.pop_back();
    if (d->heap_idx < c.heap.size()) {
        heap_update(c.heap.data(), d->heap_idx, c.heap.size());
    }
    delete d;
    c.map.erase(it);
}

static void verify(Container &c) {
    assert(c.heap.size() == c.map.size());
    for (size_t i = 0; i < c.heap.size(); ++i) {
        size_t l = heap_left(i);
        size_t r = heap_right(i);
        assert(l >= c.heap.size() || c.heap[l].val >= c.heap[i].val);
        assert(r >= c.heap.size() || c.heap[r].val >= c.heap[i].val);
        assert(*c.heap[i].ref == i);
    }
}

static void test_case(size_t sz) {
    for (uint32_t j = 0; j < 2 + sz * 2; ++j) {
        Container c;
        for (uint32_t i = 0; i < sz; ++i) {
            add(c, 1 + i * 2);
        }
        verify(c);

        add(c, j);
        verify(c);

        dispose(c);
    }

    for (uint32_t j = 0; j < sz; ++j) {
        Container c;
        for (uint32_t i = 0; i < sz; ++i) {
            add(c, i);
        }
        verify(c);

        del(c, j);
        verify(c);

        dispose(c);
    }
}

int main() {
    for (uint32_t i = 0; i < 200; ++i) {
        test_case(i);
    }
    return 0;
}
