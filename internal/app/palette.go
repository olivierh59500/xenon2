package app

// The checked release has sixteen unique colors in every level, so palette
// indices are recovered without ambiguity from the exported opaque RGBA pixels.
const paletteShaderSource = `//kage:unit pixels
package main
var Palette [16]vec4
var Mask float
var Shades float
func Fragment(dstPos vec4,srcPos vec2,color vec4) vec4 {
 pixel:=imageSrc0At(srcPos)
 for i:=0;i<16;i++ {
  if distance(pixel.rgb,Palette[i].rgb)<0.00001 {
   if Mask>=0 {
    if (int(Mask)>>i)&1!=0 {return vec4(vec3(238.0/255.0),pixel.a)}
    return vec4(vec3(0),pixel.a)
   }
   if Shades>0 {return vec4(floor(Palette[i].rgb*255.0/68.0+0.00001)*34.0/255.0,pixel.a)}
   return pixel
  }
 }
 return pixel
}
`

const sparkShaderSource = `//kage:unit pixels
package main
var Palette [16]vec4
func Fragment(dstPos vec4,srcPos vec2,color vec4) vec4 {
 pixel:=imageSrc0At(srcPos)
 local:=srcPos-imageSrc0Origin()
 x,y:=int(local.x),int(local.y)
 for index:=0;index<16;index++ {
  if distance(pixel.rgb,Palette[index].rgb)<0.00001 {
   value:=index
   if x==1&&y==1 {value=(value|5)&7}else if x==1 {value=value|7}else{value=value|5}
   return Palette[value]
  }
 }
 return pixel
}
`

// Sprite coverage is masked by terrain coverage before copying its colors.
const terrainClippedSpriteShaderSource = `//kage:unit pixels
package main
var Position vec2
func Fragment(dstPos vec4,srcPos vec2,color vec4) vec4 {
 pixel:=imageSrc0At(srcPos)
 local:=srcPos-imageSrc0Origin()
 covered:=imageSrc1At(imageSrc1Origin()+Position+local).a
 return pixel*(1-covered)
}
`

const backgroundStarShaderSource = `//kage:unit pixels
package main
func Fragment(dstPos vec4,srcPos vec2,color vec4) vec4 {
 pixel:=imageSrc0At(srcPos)
 star:=imageSrc1At(imageSrc1Origin()+srcPos-imageSrc0Origin())
 if star.a>0&&pixel.r==0&&pixel.g==0&&pixel.b==0 {return star}
 return pixel
}
`
