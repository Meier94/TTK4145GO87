#include <comedilib.h>

#include "io.h"


//in port 4
#define PORT_4_SUBDEVICE        3
#define PORT_4_CHANNEL_OFFSET   16
#define PORT_4_DIRECTION        COMEDI_INPUT
//in port 1
#define PORT_1_SUBDEVICE        2
#define PORT_1_CHANNEL_OFFSET   0
#define PORT_1_DIRECTION        COMEDI_INPUT
//out port 3
#define PORT_3_SUBDEVICE        3
#define PORT_3_CHANNEL_OFFSET   8
#define PORT_3_DIRECTION        COMEDI_OUTPUT
//out port 2
#define PORT_2_SUBDEVICE        3
#define PORT_2_CHANNEL_OFFSET   0
#define PORT_2_DIRECTION        COMEDI_OUTPUT


static comedi_t *it_g = NULL;


int io_init(void) {

    it_g = comedi_open("/dev/comedi0");

    if (it_g == NULL) {
        printf("fail");
        comedi_perror("comedi_open");
        return 0;
    }

    int status = 0;
    for (int i = 0; i < 8; i++) {
        status |= comedi_dio_config(it_g, PORT_1_SUBDEVICE, i + PORT_1_CHANNEL_OFFSET, PORT_1_DIRECTION);
        status |= comedi_dio_config(it_g, PORT_2_SUBDEVICE, i + PORT_2_CHANNEL_OFFSET, PORT_2_DIRECTION);
        status |= comedi_dio_config(it_g, PORT_3_SUBDEVICE, i + PORT_3_CHANNEL_OFFSET, PORT_3_DIRECTION);
        status |= comedi_dio_config(it_g, PORT_4_SUBDEVICE, i + PORT_4_CHANNEL_OFFSET, PORT_4_DIRECTION);
    }

    return (status == 0);
}

//values read from channels.h 02/2018
//type up = 0, down, cab 
//lights[floor][type]
static int lights[4][3] = {
    {0x09, 0x10, 0xD},
    {0x08, 0x07, 0xC},
    {0x06, 0x05, 0xB},
    {0x10, 0x04, 0xA}};

//for etasje lys
static int floorlight[4] = {0x0, 0x2, 0x1, 0x3};

static uint16_t pusheddown = 0;

int get_signals(){
    static uint16_t active = 0;
    static uint16_t updates = 0;
    unsigned int data = 0;
    comedi_dio_bitfield2(it_g, 3, 0, &data, 16);
    unsigned int data2 = 0;
    comedi_dio_bitfield2(it_g, 2, 0, &data2, 0);
    data = (data & 0xFF) | ((data2 & 0xFF) << 8);
    updates = active^(uint16_t)data;
    pusheddown = (updates&(~active));
    active = active^updates;
    if(pusheddown>0){
        return 1;
    }
    return 0;
}

static uint16_t events[16] = {0x0200, 0x0100, 0x0402, 0x0302, 0x0202, 0x0102,
                              0x0004, 0x0005, 0x0201, 0x0300, 0x0301, 0x0401,
                              0x0103, 0x0203, 0x0303, 0x0403};

uint16_t getEvent(){
    while(pusheddown) {
        uint16_t event = pusheddown & (-pusheddown);
        int index = __builtin_ctz(event);
        pusheddown &= ~(event);
        if(index == 6 || index == 7){
            continue;
        }
        return events[index];
    }
}

void set_floor_light(int floor){
    floor--;
    unsigned int data = floorlight[floor];
    comedi_dio_bitfield2(it_g, 3, 0x3, &data, 0);
    
}

void clear_all_lights(){
    unsigned int data = 0x0;
    comedi_dio_bitfield2(it_g, 3, 0xFFFF, &data, 0);
}


void set_button_light(int floor, int type, int value){
    floor--;
    comedi_dio_write(it_g, 3, lights[floor][type], value);
}

void set_door_light(int value){
    comedi_dio_write(it_g, 3, 0x3, value);
}


void set_motor(int dir) {
    if (dir == 0){
        comedi_dio_write(it_g, 3, 0xF, 0);
        comedi_data_write(it_g, 1, 0, 0, AREF_GROUND, 2800);
    }
    else if (dir == 1){
        comedi_dio_write(it_g, 3, 0xF, 1);
        comedi_data_write(it_g, 1, 0, 0, AREF_GROUND, 2800);
    }
    else{
        comedi_data_write(it_g, 1, 0, 0, AREF_GROUND, 0);
    }
}