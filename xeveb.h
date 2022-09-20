#include <stdio.h>
#include <string.h>
#include "xeveb/xeve.h"

#ifndef H_GO_XEVEB
#define H_GO_XEVEB

typedef struct xeveb_encode_result_t {
  int status;
  int nalu_type;
  int slice_type;
  unsigned char *data;
  int size;
} xeveb_encode_result_t;

int xeveb_param_set_preset_tune(XEVE_PARAM *param, unsigned char preset, unsigned char tune) {
  return xeve_param_ppt(param, XEVE_PROFILE_BASELINE, preset, tune);
}

int xeveb_param_set_input_size(XEVE_PARAM *param, int32_t width, int32_t height) {
  param->w = width;
  param->h = height;
  return 0;
}

int xeveb_param_set_framerate(XEVE_PARAM *param, int32_t fps, int32_t keyint) {
  param->fps = fps;
  param->keyint = keyint;
  return 0;
}

int xeveb_param_set_ratecontrol(XEVE_PARAM *param, int32_t rc_type) {
  param->rc_type = rc_type;
  return 0;
}

int xeveb_param_set_bitrate(XEVE_PARAM *param, int32_t bitrate) {
  param->bitrate = bitrate;
  return 0;
}

int xeveb_param_set_gop(XEVE_PARAM *param, int32_t gop_type) {
  param->closed_gop = gop_type;
  return 0;
}

int xeveb_param_set_bframes(XEVE_PARAM *param, int32_t size) {
  if (size < 1) {
    param->inter_slice_type = 1;
    return 0;
  }

  // use B frame
  param->bframes = size;
  return 0;
}

int xeveb_param_set_use_annexb(XEVE_PARAM *param, int32_t use_annexb) {
  param->use_annexb = use_annexb;
  return 0;
}

void xeveb_free_xeve_param(XEVE_PARAM *param) {
  free(param);
}

XEVE_PARAM *xeveb_default_param() {
  XEVE_PARAM *param = (XEVE_PARAM *) malloc(sizeof(XEVE_PARAM));
  if (NULL == param) {
    return NULL;
  }

  int ret = xeve_param_default(param);
  if (XEVE_FAILED(ret)) {
    xeveb_free_xeve_param(param);
    return NULL;
  }
  ret = xeve_param_ppt(param, XEVE_PROFILE_BASELINE, XEVE_PRESET_DEFAULT, XEVE_TUNE_NONE);
  if (XEVE_FAILED(ret)) {
    xeveb_free_xeve_param(param);
    return NULL;
  }

  param->closed_gop = 1;
  return param;
}

void xeveb_free_xeve(XEVE id) {
  xeve_delete(id);
}

XEVE xeveb_create(XEVE_PARAM *param, int32_t max_bitstream_buffer_size) {
  XEVE_CDSC cdsc;
  memset(&cdsc, 0, sizeof(XEVE_CDSC));
  cdsc.max_bs_buf_size = max_bitstream_buffer_size;
  memcpy(&cdsc.param, param, sizeof(XEVE_PARAM));

  int ret;
  XEVE id = xeve_create(&cdsc, &ret);
  if(XEVE_FAILED(ret)) {
    return NULL;
  }
  return id;
}

void xeveb_free_result(xeveb_encode_result_t *result) {
  if(result != NULL){
    free(result->data);
  }
  free(result);
}

void xeveb_free_bitb(XEVE_BITB *bitb) {
  if(bitb != NULL){
    free(bitb->addr);
  }
  free(bitb);
}

XEVE_BITB *xeveb_create_bitb(int32_t max_bitstream_buffer_size) {
  unsigned char *bs_buf = (unsigned char *) malloc(max_bitstream_buffer_size);
  if(NULL == bs_buf) {
    return NULL;
  }

  XEVE_BITB *bitb = (XEVE_BITB *) malloc(sizeof(XEVE_BITB));
  if(NULL == bitb) {
    free(bs_buf);
    return NULL;
  }
  memset(bitb, 0, sizeof(XEVE_BITB));
  bitb->addr = bs_buf;
  bitb->bsize = max_bitstream_buffer_size;
  return bitb;
}

void xeveb_free_imgb(XEVE_IMGB *imgb) {
  free(imgb);
}

XEVE_IMGB *xeveb_create_imgb(
  XEVE_PARAM *param,
  unsigned char *y,
  unsigned char *u,
  unsigned char *v,
  int32_t stride_y,
  int32_t stride_u,
  int32_t stride_v,
  int32_t size_y,
  int32_t size_u,
  int32_t size_v,
  uint8_t color_format,
  uint8_t bit_depth
) {
  XEVE_IMGB *imgb = (XEVE_IMGB *) malloc(sizeof(XEVE_IMGB));
  if(NULL == imgb) {
    return NULL;
  }
  memset(imgb, 0, sizeof(XEVE_IMGB));

  imgb->a[0] = y;
  imgb->a[1] = u;
  imgb->a[2] = v;
  imgb->s[0] = stride_y;
  imgb->s[1] = stride_u;
  imgb->s[2] = stride_v;
  imgb->bsize[0] = size_y;
  imgb->bsize[1] = size_u;
  imgb->bsize[2] = size_v;
  imgb->x[0] = 0;
  imgb->x[1] = 0;
  imgb->x[2] = 0;
  imgb->y[0] = 0;
  imgb->y[1] = 0;
  imgb->y[2] = 0;
  imgb->w[0] = param->w;
  imgb->h[0] = param->h;
  imgb->cs = XEVE_CS_SET(color_format, bit_depth, 0);

  switch(XEVE_CS_GET_FORMAT(param->cs)) {
  case XEVE_CF_YCBCR400:
    imgb->w[1] = imgb->w[2] = param->w;
    imgb->h[1] = imgb->h[2] = param->h;
    imgb->np = 1;
    break;
  case XEVE_CF_YCBCR420:
    imgb->w[1] = imgb->w[2] = param->w / 2;
    imgb->h[1] = imgb->h[2] = param->h / 2;
    imgb->np = 3;
    break;
  case XEVE_CF_YCBCR422:
    imgb->w[1] = imgb->w[2] = param->w / 2;
    imgb->h[1] = imgb->h[2] = param->h;
    imgb->np = 3;
    break;
  case XEVE_CF_YCBCR444:
    imgb->w[1] = imgb->w[2] = param->w;
    imgb->h[1] = imgb->h[2] = param->h;
    imgb->np = 3;
    break;
  default:
    // default YCbCr 420
    imgb->w[1] = imgb->w[2] = param->w / 2;
    imgb->h[1] = imgb->h[2] = param->h / 2;
    imgb->np = 3;
    break;
  }

  for(int i = 0; i < imgb->np; i += 1){
    imgb->aw[i] = imgb->w[i];
    imgb->ah[i] = imgb->h[i];
  }

  return imgb;
}

int xeveb_bump(XEVE id, XEVE_BITB *bitb) {
  int val  = 1;
  int size = sizeof(int);
  return xeve_config(id, XEVE_CFG_SET_FORCE_OUT, (void *)(&val), &size);
}

int xeveb_push(XEVE id, XEVE_IMGB *imgb) {
  return xeve_push(id, imgb);
}

xeveb_encode_result_t *xeveb_encode(
  XEVE id,
  XEVE_BITB *bitb
) {
  XEVE_STAT stat;
  memset(&stat, 0, sizeof(XEVE_STAT));

  int ret_encode = xeve_encode(id, bitb, &stat);
  if(XEVE_FAILED(ret_encode)) {
    return NULL;
  }

  xeveb_encode_result_t *result = (xeveb_encode_result_t *) malloc(sizeof(xeveb_encode_result_t));
  memset(result, 0, sizeof(xeveb_encode_result_t));

  if(ret_encode == XEVE_OK_OUT_NOT_AVAILABLE) {
    result->status = XEVE_OK_OUT_NOT_AVAILABLE;
    return result;
  }
  if(ret_encode == XEVE_OK_NO_MORE_FRM) {
    result->status = XEVE_OK_NO_MORE_FRM;
    return result;
  }
  if(ret_encode != XEVE_OK) {
    result->status = ret_encode;
    return result;
  }

  if(stat.write < 1) {
    result->status = XEVE_OK_OUT_NOT_AVAILABLE;
    return result;
  }

  result->data = (unsigned char*) malloc(stat.write);
  if(NULL == result->data) {
    result->status = XEVE_ERR_OUT_OF_MEMORY;
    return result;
  }
  memcpy(result->data, bitb->addr, stat.write);

  result->status = XEVE_OK;
  result->nalu_type = stat.nalu_type;
  result->slice_type = stat.stype;
  result->size = stat.write;
  return result;
}

#endif
