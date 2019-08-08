/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

  * Minimum Working Example of implementation of a C++ iterator (as well as
  * const_iterator and reverse_iterator) on your own object
  *
  * Simpliy compile with: 
  *   g++ mwe_iterator.cpp

*/
#include <cstring>
#include <iostream>
#include <vector>
#include <string>

// Simple class that we want to iterate on.
// A counter is a label and a value.
class Counter {
 public:
  Counter(const std::string &node, unsigned int counter):mNode(node), mCounter(counter) {}

  std::string getNode() const { return mNode; }
  unsigned int getCounter() const { return mCounter; }
 private:
  std::string mNode;
  unsigned int mCounter;
};

// The container, a simple vector of Counter
class CounterVector {
 public:
  // The "basic" iterator
  class iterator {
   public:
    explicit iterator(CounterVector *ptr, int idx);
    iterator operator++();    // prefix operator: ++i
    iterator operator++(int); // postfix operator: i++
    bool operator!=(const iterator & other) const;
    Counter& operator*() const;
    const Counter * operator+(size_t s) const;
    const Counter * operator->();
   private:
    CounterVector *pPtr;
    int mIdx;
  };

  // The const_iterator
  class const_iterator {
   public:
    explicit const_iterator(const CounterVector *ptr, int idx);
    const_iterator operator++();    // prefix operator: ++i
    const_iterator operator++(int); // postfix operator: i++
    bool operator!=(const const_iterator & other) const;
    Counter& operator*() const;
    const Counter* operator+(size_t s) const;
    const Counter * operator->();
   private:
    const CounterVector *pPtr;
    int mIdx;
  };

  // The reverse iterator
  class reverse_iterator {
   public:
    explicit reverse_iterator(CounterVector *ptr, int idx);
    reverse_iterator operator++();    // prefix operator: ++i
    reverse_iterator operator++(int); // postfix operator: i++
    bool operator!=(const reverse_iterator & other) const;
    Counter& operator*() const;
    const Counter* operator+(size_t s) const;
    const Counter * operator->();
   private:
    CounterVector *pPtr;
    int mIdx;
  };

  CounterVector():mCounters() {}

  void addCounter(Counter *c) {
      mCounters.push_back(c);
  }

  void addCounter(const std::string &node, unsigned int counterValue) {
      Counter *c = new Counter(node, counterValue);
      mCounters.push_back(c);
  }

  std::vector<Counter *> getCounters() const {
      return mCounters;
  }

  std::vector<std::string> getNodes() const;

  size_t size() const;
  Counter *operator[](int i);
  Counter *operator[](int i) const;

  reverse_iterator rbegin();
  reverse_iterator rend();
  const_iterator begin() const;
  const_iterator end() const;
  iterator begin();
  iterator end();

  const Counter *back() const;
  Counter *back();

 private:
  std::vector<Counter *> mCounters;
};

class TestClass {
 private:
  std::vector<CounterVector> mCounterVectors;

 public:
  TestClass();

  std::vector<CounterVector> getCounterVectors() { return mCounterVectors; }
};

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////
// iterator
////////////////////////////////////////
CounterVector::iterator::iterator(CounterVector *ptr, int idx): pPtr(ptr), mIdx(idx) {
}

CounterVector::iterator CounterVector::iterator::operator++() {
    ++mIdx;
    return *this;
}

CounterVector::iterator CounterVector::iterator::operator++(int) {
    ++mIdx;
    return *this;
}

bool CounterVector::iterator::operator!=(const iterator & other) const {
    return mIdx != other.mIdx;
}

Counter& CounterVector::iterator::operator*() const {
    Counter * c = (*pPtr)[mIdx];
    return *c;
}

const Counter * CounterVector::iterator::operator+(size_t s) const {
    Counter * c = (*pPtr)[mIdx + s];
    const Counter * const_c = const_cast<const Counter *>(c);
    return const_c;
}

const Counter * CounterVector::iterator::operator->() {
    Counter * c = (*pPtr)[mIdx];
    const Counter * const_c = const_cast<const Counter *>(c);
    return const_c;
}

////////////////////////////////////////
// const_iterator
////////////////////////////////////////
CounterVector::const_iterator::const_iterator(const CounterVector *ptr, int idx): pPtr(ptr), mIdx(idx) {
}

CounterVector::const_iterator CounterVector::const_iterator::operator++() {
    ++mIdx;
    return *this;
}

CounterVector::const_iterator CounterVector::const_iterator::operator++(int) {
    ++mIdx;
    return *this;
}

bool CounterVector::const_iterator::operator!=(const const_iterator & other) const {
    return mIdx != other.mIdx;
}

Counter& CounterVector::const_iterator::operator*() const {
    Counter * c = (*pPtr)[mIdx];
    return *c;
}

const Counter *CounterVector::const_iterator::operator+(size_t s) const {
    Counter * c = (*pPtr)[mIdx + s];
    const Counter * const_c = const_cast<const Counter *>(c);
    return const_c;
}

const Counter * CounterVector::const_iterator::operator->() {
    Counter * c = (*pPtr)[mIdx];
    const Counter * const_c = const_cast<const Counter *>(c);
    return const_c;
}

////////////////////////////////////////
// reverse_iterator
////////////////////////////////////////
CounterVector::reverse_iterator::reverse_iterator(CounterVector *ptr, int idx): pPtr(ptr), mIdx(idx) {
}

CounterVector::reverse_iterator CounterVector::reverse_iterator::operator++() {
    --mIdx;
    return *this;
}

CounterVector::reverse_iterator CounterVector::reverse_iterator::operator++(int) {
    --mIdx;
    return *this;
}

bool CounterVector::reverse_iterator::operator!=(const reverse_iterator & other) const {
    return mIdx != other.mIdx;
}

Counter& CounterVector::reverse_iterator::operator*() const {
    Counter * c = (*pPtr)[mIdx];
    return *c;
}

const Counter *CounterVector::reverse_iterator::operator+(size_t s) const {
    Counter * c = (*pPtr)[mIdx - s];
    const Counter * const_c = const_cast<const Counter *>(c);
    return const_c;
}

const Counter * CounterVector::reverse_iterator::operator->() {
    Counter * c = (*pPtr)[mIdx];
    const Counter * const_c = const_cast<const Counter *>(c);
    return const_c;
}
////////////////////////////////////////
// CounterVector class methods
////////////////////////////////////////

size_t CounterVector::size() const {
    return mCounters.size();
}

Counter *CounterVector::operator[](int i) {
    return mCounters[i];
}

Counter *CounterVector::operator[](int i) const {
    return mCounters[i];
}

std::vector<std::string> CounterVector::getNodes() const {
    std::vector<std::string> v;
    for (auto &c: mCounters) {
        v.push_back(c->getNode());
    }
    return v;
}

CounterVector::iterator CounterVector::begin() {
    return CounterVector::iterator(this, 0);
}

CounterVector::iterator CounterVector::end() {
    CounterVector::iterator itr = CounterVector::iterator(this, mCounters.size());
    return itr;
}

CounterVector::reverse_iterator CounterVector::rbegin() {
    CounterVector::reverse_iterator itr = CounterVector::reverse_iterator(this, mCounters.size() - 1);
    return itr;
}

CounterVector::reverse_iterator CounterVector::rend() {
    return CounterVector::reverse_iterator(this, -1);
}

CounterVector::const_iterator CounterVector::begin() const {
    return CounterVector::const_iterator(this, 0);
}

CounterVector::const_iterator CounterVector::end() const {
    CounterVector::const_iterator itr = CounterVector::const_iterator(this, mCounters.size());
    return itr;
}

const Counter *CounterVector::back() const {
    return mCounters.back();
}

Counter *CounterVector::back() {
    return mCounters.back();
}

//////////////////////////////////////////////////////////////////////
TestClass::TestClass() {
    CounterVector counter_for_new_resource;
    Counter *counter = new Counter ("A", 1);
    counter_for_new_resource.addCounter(counter);
    counter = new Counter ("B", 1);
    counter_for_new_resource.addCounter(counter);
    mCounterVectors.push_back(counter_for_new_resource);

    CounterVector counter_for_new_resource2;
    counter = new Counter ("D", 1);
    counter_for_new_resource2.addCounter(counter);
    counter = new Counter ("C", 2);
    counter_for_new_resource2.addCounter(counter);
    counter = new Counter ("E", 1);
    counter_for_new_resource2.addCounter(counter);
    mCounterVectors.push_back(counter_for_new_resource2);
}

int main(int argc, char**argv) {
    TestClass *c = new TestClass();;

    std::string s_iterator1 = "auto iterator: ";
    std::string s_iterator2 = "iterator: ";
    std::string s_const_iterator = "const_iterator: ";
    std::string s_reverse_iterator = "reverse_iterator: ";

    // sample iterator on a vector of integers
    std::cerr << "vector<int> iterator" << std::endl;
    std::vector<int> sample_vector = {1, 2, 3, 4, 5};
    for (std::vector<int>::iterator it = sample_vector.begin(); it != sample_vector.end(); ++it) {
	std::cerr << "*it=" << *it << std::endl;
	// Does not crash, g++ output 0 when accessing an out-o-range iterators
	std::cerr << "*(it + 1)=" << *(it + 1) << std::endl;
    }

    // sample iterator on a vector of string
    std::cerr << "vector<string> iterator" << std::endl;
    std::vector<std::string> sample_string_vector = {"a", "b", "c", "d", "e"};
    for (std::vector<std::string>::iterator it = sample_string_vector.begin(); it != sample_string_vector.end(); ++it) {
	std::cerr << "*it=" << *it << std::endl;
	// No exception is raised when trying to access an iterator
	// out of range, it just segfaults
	/*
	try {
	    std::cerr << "*(it + 1)=" << *(it + 1) << std::endl;
	} catch(const std::exception &e) {
	    std::cerr << e.what() << std::endl;
	    }*/
	
    }

    std::cerr << "CounterVector iterator" << std::endl;
    for (auto &all_v : c->getCounterVectors()) {
	// std::cerr << __func__ << ":" << __LINE__ << std::endl;
        // std::cerr << "type of all_v=" << typeid(all_v).name() << std::endl;
        // std::cerr << "all_v.size=" << all_v.size() << std::endl;
        for (auto v : all_v) {
            // std::cerr << "type of v=" << typeid(v).name() << std::endl;
            // std::cerr << std::hex << &v << std::dec << std::endl;
            // std::cerr << "v->getNode=" << v.getNode() << std::endl;
            // std::cerr << "v->getCounter=" << v.getCounter() << std::endl;
            s_iterator1 += v.getNode() + "(" + std::to_string(v.getCounter()) + "),";
        }
        // std::cerr << "toto" << std::endl;
        // CounterVector::iterator it = all_v.begin();
        // std::cerr << "begin="<< std::hex << &it << std::dec << std::endl;
        // it = all_v.end();
        // std::cerr << "end=" << std::hex << &it << std::dec <<
        // std::endl;
	unsigned int i = 0;
        for (CounterVector::iterator itr= all_v.begin(); itr != all_v.end(); ++itr) {
            // std::cerr << "itr.getNode=" << (*itr).getNode() << std::endl;
            // std::cerr << "itr.getCounter=" << (*itr).getCounter() << std::endl;
            s_iterator2 += (*itr).getNode() + "(" + std::to_string((*itr).getCounter()) + "),";
	    // std::cerr << "i=" << i << std::endl;
	    // std::cerr << "size=" << all_v.size() << std::endl;
	    if (i + 1 <= all_v.size() - 1) {
		// Testing operator+
		std::cerr << (itr + 1)->getNode() << std::endl;
              }
	    ++i;
        }

        const CounterVector* v2 = const_cast<const CounterVector *>(&all_v);
        for (CounterVector::const_iterator itr = v2->begin(); itr != v2->end(); ++itr) {
            // s_const_iterator += (*itr).getNode() + "(" + std::to_string((*itr).getCounter()) + "),";
            s_const_iterator += itr->getNode() + "(" + std::to_string(itr->getCounter()) + "),";
        }
        for (CounterVector::reverse_iterator itr = all_v.rbegin(); itr != all_v.rend(); ++itr) {
            s_reverse_iterator += (*itr).getNode() + "(" + std::to_string((*itr).getCounter()) + "),";
        }
    }
    std::cerr << s_iterator1 << std::endl;
    std::cerr << s_iterator2 << std::endl;
    std::cerr << s_const_iterator << std::endl;
    std::cerr << s_reverse_iterator << std::endl;
}
