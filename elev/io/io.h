#pragma once
#include <stdint.h>
// Returns 0 on init failure
int io_init(void);

int get_signals();

void set_button_light(int floor, int type, int value);

void set_floor_light(int floor);

void clear_all_lights();

void set_door_light(int value);

void set_motor(int dir);

uint16_t getEvent();

