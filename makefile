CXXFLAGS  = -std=c++11 -g -O3 -Wall 
LDLIBS  = -lpthread


SOURCES = $(wildcard *.cpp)
OBJECTS = $(patsubst %.cpp,%.o,$(SOURCES))

TARGET  = hornet

all : $(TARGET)

$(TARGET) : $(OBJECTS)
	$(CXX) $(CXXFLAGS) -o $@ $^  $(LDLIBS)

clean :
	rm -rf $(TARGET) *.o 
