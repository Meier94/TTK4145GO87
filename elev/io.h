#pragma once
#include <stdint.h>
// Returns 0 on init failure
int io_init(void);

void io_set_bit(int channel);
void io_clear_bit(int channel);

int io_read_bit(int channel);

int io_read_analog(int channel);
void io_write_analog(int channel, int value);

int get_signals();

void set_button_light(int floor, int type, int value);

void set_floor_light(int floor);

void clear_all_lights();

uint16_t getEvent();

void set_motor(int dir);
